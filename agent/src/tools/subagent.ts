import { tool } from 'ai'
import { z } from 'zod'
import { AgentAction, createAgent } from '../agent'
import { BaseModelConfig } from '../types'
import { AuthFetcher } from '..'

export interface SubagentToolParams extends BaseModelConfig {
  fetch: AuthFetcher
  braveApiKey?: string
  braveBaseUrl?: string
}

export const getSubagentTools = ({ fetch, apiKey, baseUrl, model, clientType, braveApiKey, braveBaseUrl }: SubagentToolParams) => {
  const listSubagents = tool({
    description: 'List subagents for current user',
    inputSchema: z.object({}),
    execute: async () => {
      const response = await fetch('/subagents', { method: 'GET' })
      return response.json()
    },
  })

  const createSubagent = tool({
    description: 'Create a new subagent',
    inputSchema: z.object({
      name: z.string(),
      description: z.string(),
    }),
    execute: async ({ name, description }) => {
      const response = await fetch('/subagents', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, description }),
      })
      return response.json()
    },
  })

  const deleteSubagent = tool({
    description: 'Delete a subagent by id',
    inputSchema: z.object({
      id: z.string().describe('Subagent ID'),
    }),
    execute: async ({ id }) => {
      const response = await fetch(`/subagents/${id}`, { method: 'DELETE' })
      return response.status === 204 ? { success: true } : response.json()
    },
  })

  const querySubagent = tool({
    description: 'Query a subagent',
    inputSchema: z.object({
      name: z.string(),
      query: z.string().describe('The prompt to ask the subagent to do.'),
    }),
    execute: async ({ name, query }) => {
      const listResponse = await fetch('/subagents', { method: 'GET' })
      const listPayload = await listResponse.json()
      const items = Array.isArray(listPayload?.items) ? listPayload.items : []
      const target = items.find((item: { name?: string }) => item?.name === name)
      if (!target?.id) {
        throw new Error(`subagent not found: ${name}`)
      }
      const contextResponse = await fetch(`/subagents/${target.id}/context`, { method: 'GET' })
      const contextPayload = await contextResponse.json()
      const contextMessages = Array.isArray(contextPayload?.messages) ? contextPayload.messages : []
      const { askAsSubagent } = createAgent({
        apiKey,
        baseUrl,
        model,
        clientType,
        braveApiKey,
        braveBaseUrl,
        allowed: [
          AgentAction.WebSearch,
        ]
      })
      const result = await askAsSubagent({
        messages: contextMessages,
        query,
      }, { name, description: target.description })
      const updatedMessages = [...contextMessages, ...result.messages]
      await fetch(`/subagents/${target.id}/context`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ messages: updatedMessages }),
      })
      return {
        success: true,
        result: result.messages[result.messages.length - 1].content,
      }
    },
  })

  return {
    'list_subagents': listSubagents,
    'create_subagent': createSubagent,
    'delete_subagent': deleteSubagent,
    'query_subagent': querySubagent,
  }
}