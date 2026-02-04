import { tool } from 'ai'
import { z } from 'zod'
import { AuthFetcher } from '..'
import type { ToolContext } from '../agent'

export type MessageToolParams = {
  fetch: AuthFetcher
  toolContext?: ToolContext
}

const SendMessageSchema = z.object({
  bot_id: z.string().optional(),
  platform: z.string().optional(),
  target: z.string().optional(),
  to_user_id: z.string().optional(),
  message: z.string(),
})

export const getMessageTools = ({ fetch, toolContext }: MessageToolParams) => {
  const sendMessage = tool({
    description: 'Send a message to a channel or session',
    inputSchema: SendMessageSchema,
    execute: async (payload) => {
      const botId = (payload.bot_id ?? toolContext?.botId ?? '').trim()
      const platform = (payload.platform ?? toolContext?.currentPlatform ?? '').trim()
      const replyTarget = (toolContext?.replyTarget ?? '').trim()
      const target = (payload.target ?? replyTarget).trim()
      const toUserID = (payload.to_user_id ?? '').trim()
      if (!botId) {
        throw new Error('bot_id is required')
      }
      if (!platform) {
        throw new Error('platform is required')
      }
      if (!target && !toUserID && !toolContext?.sessionToken) {
        throw new Error('target or to_user_id is required')
      }
      // Use session token if available and no explicit to_user_id specified
      // This allows replying to current session without needing explicit auth
      const useSessionToken = !!toolContext?.sessionToken && !toUserID
      console.log('[Tool] send_message', {
        botId,
        platform,
        target: target || undefined,
        toUserID: toUserID || undefined,
        replyTarget,
        useSessionToken,
      })
      const body: Record<string, unknown> = { message: payload.message }
      if (!useSessionToken) {
        if (target) {
          body.to = target
        }
        if (toUserID) {
          body.to_user_id = toUserID
        }
      }
      const url = useSessionToken
        ? `/bots/${botId}/channel/${platform}/send_session`
        : `/bots/${botId}/channel/${platform}/send`
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (useSessionToken && toolContext?.sessionToken) {
        headers.Authorization = `Bearer ${toolContext.sessionToken}`
      }
      const response = await fetch(url, {
        method: 'POST',
        headers,
        body: JSON.stringify(body),
      })
      const result = await response.json()
      return {
        ...result,
        instruction: 'Message delivered successfully. You have completed your response. Please STOP now and do not call any more tools.',
      }
    },
  })

  return {
    'send_message': sendMessage,
  }
}
