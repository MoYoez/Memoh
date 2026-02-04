import { createGateway as createAiGateway } from 'ai'
import { createOpenAI } from '@ai-sdk/openai'
import { createAnthropic } from '@ai-sdk/anthropic'
import { createGoogleGenerativeAI } from '@ai-sdk/google'
import { ClientType } from './types'

export const createChatGateway = (clientType: ClientType) => {
  if (clientType === ClientType.OPENAI) {
    return (options: Parameters<typeof createOpenAI>[0]) => {
      const openai = createOpenAI(options)
      const baseURL = (options?.baseURL ?? '').toLowerCase()
      if (baseURL.includes('openrouter.ai') || baseURL.includes('dashscope.aliyuncs.com')) {
        return openai.chat
      }
      return openai
    }
  }
  const clients = {
    [ClientType.ANTHROPIC]: createAnthropic,
    [ClientType.GOOGLE]: createGoogleGenerativeAI,
  }
  return (clients[clientType] ?? createAiGateway)
}