import { generateText, ModelMessage, stepCountIs, streamText, TextStreamPart, ToolSet } from 'ai'
import { createChatGateway } from './gateway'
import { BaseModelConfig, Schedule } from './types'
import { system, schedule } from './prompts'
import { AuthFetcher } from './index'
import { getScheduleTools } from './tools/schedule'
import { getWebTools } from './tools/web'
import { subagentSystem } from './prompts/subagent'
import { getSubagentTools } from './tools/subagent'

export enum AgentAction {
  WebSearch = 'web_search',
  Message = 'message',
  Subagent = 'subagent',
  Schedule = 'schedule',
  Skill = 'skill',
}

export interface AgentParams extends BaseModelConfig {
  locale?: Intl.LocalesArgument
  language?: string
  maxSteps?: number
  maxContextLoadTime?: number
  platforms?: string[]
  currentPlatform?: string
  braveApiKey?: string
  braveBaseUrl?: string
  allowed?: AgentAction[]
}

export interface AgentInput {
  messages: ModelMessage[]
  query: string
}

export interface AgentResult {
  messages: ModelMessage[]
}

export const createAgent = (
  params: AgentParams,
  fetcher: AuthFetcher = fetch,
) => {
  const gateway = createChatGateway(params.clientType)
  const messages: ModelMessage[] = []

  const allowedActions = params.allowed
    ?? Object.values(AgentAction)

  const maxSteps = params.maxSteps ?? 50

  const getTools = () => {
    const tools: ToolSet = {}

    if (allowedActions.includes(AgentAction.Schedule)) {
      const scheduleTools = getScheduleTools({ fetch: fetcher })
      Object.assign(tools, scheduleTools)
    }

    if (params.braveApiKey && allowedActions.includes(AgentAction.WebSearch)) {
      const webTools = getWebTools({
        braveApiKey: params.braveApiKey,
        braveBaseUrl: params.braveBaseUrl,
      })
      Object.assign(tools, webTools)
    }

    if (allowedActions.includes(AgentAction.Subagent)) {
      const subagentTools = getSubagentTools({
        fetch: fetcher,
        apiKey: params.apiKey,
        baseUrl: params.baseUrl,
        model: params.model,
        clientType: params.clientType,
        braveApiKey: params.braveApiKey,
        braveBaseUrl: params.braveBaseUrl,
      })
      Object.assign(tools, subagentTools)
    }
    
    return tools
  }

  const generateSystem = () => {
    return system({
      date: new Date(),
      locale: params.locale,
      language: params.language,
      maxContextLoadTime: params.maxContextLoadTime ?? 1550,
      platforms: params.platforms ?? [],
      currentPlatform: params.currentPlatform,
    })
  }

  const ask = async (input: AgentInput): Promise<AgentResult> => {
    messages.push(...input.messages)
    const user: ModelMessage = {
      role: 'user',
      content: input.query,
    }
    messages.push(user)
    const { response } = await generateText({
      model: gateway({
        apiKey: params.apiKey,
        baseURL: params.baseUrl,
      })(params.model),
      system: generateSystem(),
      stopWhen: stepCountIs(maxSteps),
      messages,
      tools: getTools(),
    })
    return {
      messages: [user, ...response.messages],
    }
  }

  const askAsSubagent = async (
    input: AgentInput,
    options: {
      name: string
      description?: string
    }
  ): Promise<AgentResult> => {
    messages.push(...input.messages)
    const user: ModelMessage = {
      role: 'user',
      content: input.query,
    }
    messages.push(user)
    const { response } = await generateText({
      model: gateway({
        apiKey: params.apiKey,
        baseURL: params.baseUrl,
      })(params.model),
      system: subagentSystem({ date: new Date(), name: options.name, description: options.description }),
      stopWhen: stepCountIs(maxSteps),
      messages,
      prepareStep: () => {
        return {
          system: subagentSystem({ date: new Date(), name: options.name, description: options.description }),
        }
      },
      tools: getTools(),
    })
    return {
      messages: [user, ...response.messages],
    }
  }

  async function* stream(input: AgentInput): AsyncGenerator<TextStreamPart<ToolSet>, AgentResult> {
    messages.push(...input.messages)
    const user: ModelMessage = {
      role: 'user',
      content: input.query,
    }
    messages.push(user)
    const { response, fullStream } = streamText({
      model: gateway({
        apiKey: params.apiKey,
        baseURL: params.baseUrl,
      })(params.model),
      system: generateSystem(),
      stopWhen: stepCountIs(maxSteps),
      messages,
      tools: getTools(),
    })
    for await (const event of fullStream) {
      yield event
    }
    return {
      messages: [user, ...(await response).messages],
    }
  }

  const triggerSchedule = async (
    input: AgentInput,
    scheduleData: Schedule
  ): Promise<AgentResult> => {
    messages.push(...input.messages)
    const user: ModelMessage = {
      role: 'user',
      content: schedule({
        schedule: scheduleData,
        locale: params.locale,
        date: new Date(),
      }),
    }
    messages.push(user)
    const { response } = await generateText({
      model: gateway({
        apiKey: params.apiKey,
        baseURL: params.baseUrl,
      })(params.model),
      system: generateSystem(),
      stopWhen: stepCountIs(maxSteps),
      messages,
      tools: getTools(),
    })
    return {
      messages: [user, ...response.messages],
    }
  }

  return {
    ask,
    stream,
    triggerSchedule,
    askAsSubagent,
  }
}