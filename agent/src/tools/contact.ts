import { tool } from 'ai'
import { z } from 'zod'
import { AuthFetcher } from '..'
import type { ToolContext } from '../agent'

export type ContactToolParams = {
  fetch: AuthFetcher
  toolContext?: ToolContext
}

const ContactID = z.string().min(1)

const ContactCreateSchema = z.object({
  bot_id: z.string().optional(),
  display_name: z.string().optional(),
  alias: z.string().optional(),
  tags: z.array(z.string()).optional(),
  status: z.string().optional(),
  metadata: z.object({}).passthrough().optional(),
})

const ContactUpdateSchema = z.object({
  bot_id: z.string().optional(),
  contact_id: ContactID,
  display_name: z.string().optional(),
  alias: z.string().optional(),
  tags: z.array(z.string()).optional(),
  status: z.string().optional(),
  metadata: z.object({}).passthrough().optional(),
})

const ContactSearchSchema = z.object({
  bot_id: z.string().optional(),
  query: z.string().optional(),
})

const ContactBindTokenSchema = z.object({
  bot_id: z.string().optional(),
  contact_id: ContactID,
  target_platform: z.string().optional(),
  target_external_id: z.string().optional(),
  ttl_seconds: z.number().optional(),
})

const ContactBindSchema = z.object({
  bot_id: z.string().optional(),
  contact_id: ContactID,
  platform: z.string(),
  external_id: z.string(),
  bind_token: z.string(),
})

export const getContactTools = ({ fetch, toolContext }: ContactToolParams) => {
  const resolveBotId = (botId?: string) => (botId ?? toolContext?.botId ?? '').trim()

  const contactSearch = tool({
    description: 'Search contacts by name or alias',
    inputSchema: ContactSearchSchema,
    execute: async (payload) => {
      const botId = resolveBotId(payload.bot_id)
      if (!botId) {
        throw new Error('bot_id is required')
      }
      const query = (payload.query ?? '').trim()
      const url = query
        ? `/bots/${botId}/contacts?q=${encodeURIComponent(query)}`
        : `/bots/${botId}/contacts`
      const response = await fetch(url)
      return response.json()
    },
  })

  const contactCreate = tool({
    description: 'Create a contact',
    inputSchema: ContactCreateSchema,
    execute: async (payload) => {
      const botId = resolveBotId(payload.bot_id)
      if (!botId) {
        throw new Error('bot_id is required')
      }
      const response = await fetch(`/bots/${botId}/contacts`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          display_name: payload.display_name,
          alias: payload.alias,
          tags: payload.tags,
          status: payload.status,
          metadata: payload.metadata,
        }),
      })
      return response.json()
    },
  })

  const contactUpdate = tool({
    description: 'Update a contact',
    inputSchema: ContactUpdateSchema,
    execute: async (payload) => {
      const botId = resolveBotId(payload.bot_id)
      if (!botId) {
        throw new Error('bot_id is required')
      }
      const response = await fetch(`/bots/${botId}/contacts/${payload.contact_id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          display_name: payload.display_name,
          alias: payload.alias,
          tags: payload.tags,
          status: payload.status,
          metadata: payload.metadata,
        }),
      })
      return response.json()
    },
  })

  const contactBindToken = tool({
    description: 'Issue a one-time bind token for a contact',
    inputSchema: ContactBindTokenSchema,
    execute: async (payload) => {
      const botId = resolveBotId(payload.bot_id)
      if (!botId) {
        throw new Error('bot_id is required')
      }
      const response = await fetch(`/bots/${botId}/contacts/${payload.contact_id}/bind_token`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          target_platform: payload.target_platform,
          target_external_id: payload.target_external_id,
          ttl_seconds: payload.ttl_seconds,
        }),
      })
      return response.json()
    },
  })

  const contactBind = tool({
    description: 'Bind a contact to a platform identity using a bind token',
    inputSchema: ContactBindSchema,
    execute: async (payload) => {
      const botId = resolveBotId(payload.bot_id)
      if (!botId) {
        throw new Error('bot_id is required')
      }
      const response = await fetch(`/bots/${botId}/contacts/${payload.contact_id}/bind`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          platform: payload.platform,
          external_id: payload.external_id,
          bind_token: payload.bind_token,
        }),
      })
      return response.json()
    },
  })

  return {
    'contact_search': contactSearch,
    'contact_create': contactCreate,
    'contact_update': contactUpdate,
    'contact_bind_token': contactBindToken,
    'contact_bind': contactBind,
  }
}
