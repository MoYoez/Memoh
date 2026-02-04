import { generateText, ModelMessage, stepCountIs, streamText, TextStreamPart, ToolChoice, ToolSet } from 'ai'
import { createChatGateway } from './gateway'
import { AgentSkill, BaseModelConfig, Schedule } from './types'
import { system, schedule } from './prompts'
import { AuthFetcher } from './index'
import { getScheduleTools } from './tools/schedule'
import { getWebTools } from './tools/web'
import { subagentSystem } from './prompts/subagent'
import { getSubagentTools } from './tools/subagent'
import { getSkillTools } from './tools/skill'
import { getMemoryTools } from './tools/memory'
import { getMessageTools } from './tools/message'
import { getContactTools } from './tools/contact'

export enum AgentAction {
  WebSearch = 'web_search',
  Message = 'message',
  Contact = 'contact',
  Subagent = 'subagent',
  Schedule = 'schedule',
  Skill = 'skill',
  Memory = 'memory',
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
  skills?: AgentSkill[]
  useSkills?: string[]
  allowed?: AgentAction[]
  toolContext?: ToolContext
  toolChoice?: unknown
}

export interface AgentInput {
  messages: ModelMessage[]
  query: string
}

export interface AgentResult {
  messages: ModelMessage[]
  skills: string[]
}

export interface ToolContext {
  botId?: string
  sessionId?: string
  currentPlatform?: string
  replyTarget?: string
  sessionToken?: string
  contactId?: string
  contactName?: string
  contactAlias?: string
  userId?: string
}

const withToolLogging = (tools: ToolSet): ToolSet => {
  const wrapped: ToolSet = {}
  for (const [name, entry] of Object.entries(tools)) {
    const tool = entry as {
      execute?: (input: unknown) => Promise<unknown>
    }
    if (!tool?.execute) {
      wrapped[name] = entry
      continue
    }
    const wrappedTool = {
      ...(entry as Record<string, unknown>),
      execute: async (input: unknown) => {
        console.log('[Tool] call', { name, input })
        try {
          const result = await tool.execute?.(input)
          console.log('[Tool] result', { name })
          return result
        } catch (error) {
          console.error('[Tool] error', { name, error })
          throw error
        }
      },
    }
    wrapped[name] = wrappedTool as unknown as ToolSet[string]
  }
  return wrapped
}

export const createAgent = (
  params: AgentParams,
  fetcher: AuthFetcher = fetch,
) => {
  const gateway = createChatGateway(params.clientType)
  const messages: ModelMessage[] = []
  const enabledSkills: AgentSkill[] = params.skills ?? []
  enabledSkills.push(
    ...params.useSkills?.map((name) => params.skills?.find((s) => s.name === name)
  ).filter((s) => s !== undefined) ?? [])

  const allowedActions = params.allowed
    ?? Object.values(AgentAction)

  const maxSteps = params.maxSteps ?? 50

  const getTools = () => {
    const tools: ToolSet = {}

    if (allowedActions.includes(AgentAction.Skill)) {
      const skillTools = getSkillTools({
        skills: params.skills ?? [],
        useSkill: (skill) => {
          if (enabledSkills.some((s) => s.name === skill.name)) {
            return
          }
          enabledSkills.push(skill)
        }
      })
      Object.assign(tools, skillTools)
    }

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

    if (allowedActions.includes(AgentAction.Memory)) {
      const memoryTools = getMemoryTools({ fetch: fetcher })
      Object.assign(tools, memoryTools)
    }

    if (allowedActions.includes(AgentAction.Message)) {
      const messageTools = getMessageTools({
        fetch: fetcher,
        toolContext: params.toolContext,
      })
      Object.assign(tools, messageTools)
    }

    if (allowedActions.includes(AgentAction.Contact)) {
      const contactTools = getContactTools({
        fetch: fetcher,
        toolContext: params.toolContext,
      })
      Object.assign(tools, contactTools)
    }
    
    return withToolLogging(tools)
  }

  const generateSystem = () => {
    return system({
      date: new Date(),
      locale: params.locale,
      language: params.language,
      maxContextLoadTime: params.maxContextLoadTime ?? 1550,
      platforms: params.platforms ?? [],
      currentPlatform: params.currentPlatform,
      skills: params.skills ?? [],
      enabledSkills,
      toolContext: params.toolContext,
    })
  }

  const shouldForceAutoToolChoice = (error: unknown) => {
    const message = error instanceof Error
      ? error.message
      : String(error ?? '')
    if (
      message.includes('Tool choice must be auto')
      || message.includes('tool_choice')
      || message.includes('No endpoints found that support the provided')
    ) {
      return true
    }
    if (error instanceof Error && error.cause) {
      const causeMessage = error.cause instanceof Error
        ? error.cause.message
        : String(error.cause)
      return causeMessage.includes('Tool choice must be auto')
    }
    return false
  }

  const buildCallSettings = (toolChoice?: unknown) => {
    const tools = getTools()
    console.log('[Agent] tools available:', Object.keys(tools))
    return {
      model: gateway({
        apiKey: params.apiKey,
        baseURL: params.baseUrl,
      })(params.model),
      system: generateSystem(),
      stopWhen: stepCountIs(maxSteps),
      messages,
      prepareStep: () => {
        return {
          system: generateSystem(),
        }
      },
      tools,
      toolChoice: toolChoice as ToolChoice<ToolSet> | undefined,
    }
  }

  const ask = async (input: AgentInput): Promise<AgentResult> => {
    messages.push(...input.messages)
    const user: ModelMessage = {
      role: 'user',
      content: input.query,
    }
    messages.push(user)
    let response
    try {
      const result = await generateText(buildCallSettings(params.toolChoice))
      response = result.response
    } catch (error) {
      if (params.toolChoice && shouldForceAutoToolChoice(error)) {
        console.warn('[Chat] toolChoice rejected, fallback to auto')
        const result = await generateText(buildCallSettings('auto'))
        response = result.response
      } else {
        throw error
      }
    }
    return {
      messages: [user, ...response.messages],
      skills: enabledSkills.map((s) => s.name),
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
      skills: enabledSkills.map((s) => s.name),
    }
  }

  async function* stream(input: AgentInput): AsyncGenerator<TextStreamPart<ToolSet>, AgentResult> {
    messages.push(...input.messages)
    const user: ModelMessage = {
      role: 'user',
      content: input.query,
    }
    messages.push(user)
    let response
    let fullStream
    try {
      const result = streamText(buildCallSettings(params.toolChoice))
      response = result.response
      fullStream = result.fullStream
    } catch (error) {
      if (params.toolChoice && shouldForceAutoToolChoice(error)) {
        console.warn('[Chat] toolChoice rejected, fallback to auto')
        const result = streamText(buildCallSettings('auto'))
        response = result.response
        fullStream = result.fullStream
      } else {
        throw error
      }
    }
    for await (const event of fullStream) {
      yield event
    }
    return {
      messages: [user, ...(await response).messages],
      skills: enabledSkills.map((s) => s.name),
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
      prepareStep: () => {
        return {
          system: generateSystem(),
        }
      },
      tools: getTools(),
    })
    return {
      messages: [user, ...response.messages],
      skills: enabledSkills.map((s) => s.name),
    }
  }

  return {
    ask,
    stream,
    triggerSchedule,
    askAsSubagent,
  }
}