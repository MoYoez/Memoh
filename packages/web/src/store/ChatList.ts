import { defineStore } from 'pinia'
import { reactive, ref } from 'vue'
import type { user, robot } from '@memoh/shared'
import request from '@/utils/request'

export const useChatList= defineStore('chatList', () => {
  const chatList = reactive<(((user | robot)))[]>([])
  const loading=ref(false)
  const botId = ref<string | null>(null)
  const sessionId = ref<string | null>(null)
  const streamAbort = ref<(() => void) | null>(null)
  const add = (chatItem: user | robot) => {
    chatList.push(chatItem)
  }
  const nextId = () => `${Date.now()}-${Math.floor(Math.random() * 1000)}`

  const addUserMessage = (text: string) => {
    add({
      description: text,
      time: new Date(),
      action: 'user',
      id: nextId(),
    })
  }

  const addRobotMessage = (text: string) => {
    add({
      description: text,
      time: new Date(),
      action: 'robot',
      id: nextId(),
      type: 'Memoh Agent',
      state: 'complete',
    })
  }

  const extractTextFromEvent = (payload: string) => {
    try {
      const event = JSON.parse(payload)
      if (typeof event === 'string') return event
      if (typeof event?.text === 'string') return event.text
      if (typeof event?.content === 'string') return event.content
      if (typeof event?.data === 'string') return event.data
      if (typeof event?.data?.text === 'string') return event.data.text
      return null
    } catch {
      return payload
    }
  }

  const startStream = async (bot: string, session: string) => {
    if (streamAbort.value) {
      streamAbort.value()
      streamAbort.value = null
    }
    const controller = new AbortController()
    streamAbort.value = () => controller.abort()
    const token = localStorage.getItem('token') ?? ''
    const resp = await fetch(`/api/bots/${bot}/web/sessions/${session}/stream`, {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${token}`,
      },
      signal: controller.signal,
    }).catch(() => null)
    if (!resp || !resp.ok || !resp.body) {
      return
    }
    const reader = resp.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''
    while (true) {
      const { value, done } = await reader.read()
      if (done) break
      buffer += decoder.decode(value, { stream: true })
      let idx
      while ((idx = buffer.indexOf('\n')) >= 0) {
        const line = buffer.slice(0, idx).trim()
        buffer = buffer.slice(idx + 1)
        if (!line.startsWith('data:')) continue
        const payload = line.slice(5).trim()
        if (!payload || payload === '[DONE]') continue
        const text = extractTextFromEvent(payload)
        if (text) {
          addRobotMessage(text)
        }
      }
    }
  }

  const ensureSession = async () => {
    if (botId.value && sessionId.value) {
      return
    }
    const botResp = await request({
      url: '/bots',
      method: 'GET',
    })
    const bots = botResp?.data?.items ?? []
    if (!bots.length) {
      throw new Error('No bots found')
    }
    botId.value = botId.value ?? bots[0].id
    const sessionResp = await request({
      url: `/bots/${botId.value}/web/sessions`,
      method: 'POST',
    })
    sessionId.value = sessionResp?.data?.session_id
    if (botId.value && sessionId.value) {
      void startStream(botId.value, sessionId.value)
    }
  }

  const sendMessage = async (text: string) => {
    const trimmed = text.trim()
    if (!trimmed) return
    loading.value = true
    try {
      addUserMessage(trimmed)
      await ensureSession()
      if (!botId.value || !sessionId.value) {
        throw new Error('Session not ready')
      }
      await request({
        url: `/bots/${botId.value}/web/sessions/${sessionId.value}/messages`,
        method: 'POST',
        data: { text: trimmed },
      })
    } finally {
      loading.value = false
    }
  }

  return {
    chatList,
    add,
    loading,
    sendMessage,
  }
})