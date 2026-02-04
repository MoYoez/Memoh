import { Elysia } from 'elysia'
import { chatModule } from './modules/chat'
import { corsMiddleware } from './middlewares/cors'
import { errorMiddleware } from './middlewares/error'
import { loadConfig } from './config'
import { join } from 'path'

const config = loadConfig('../config.toml')

export type AuthFetcher = (url: string, options?: RequestInit) => Promise<Response>
export const createAuthFetcher = (bearer: string | undefined): AuthFetcher => {
  return async (url: string, options?: RequestInit) => {
    const requestOptions = options ?? {}
    const headers = new Headers(requestOptions.headers || {})
    if (bearer) {
      headers.set('Authorization', `Bearer ${bearer}`)
    }
    let baseUrl = ''
    if (!baseUrl) {
      baseUrl = 'http://127.0.0.1'
    }
    if (typeof config.server.addr === 'string' && config.server.addr.startsWith(':')) {
      baseUrl = `http://127.0.0.1${config.server.addr}`
    }
    return await fetch(join(baseUrl, url), {
      ...requestOptions,
      headers,
    })
  }
}

const app = new Elysia()
  .use(corsMiddleware)
  .use(errorMiddleware)
  .use(chatModule)
  .listen({
    port: config.agent_gateway.port ?? 8081,
    hostname: config.agent_gateway.host ?? '127.0.0.1',
  })

console.log(
  `Agent Gateway is running at ${app.server?.hostname}:${app.server?.port}`
)
