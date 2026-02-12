import { client } from '@memoh/sdk/client'
import { useQuery, useMutation, useQueryCache } from '@pinia/colada'

// ---- Types ----

export interface MCPListItem {
  id: string
  type: string
  name: string
  config: {
    cwd: string
    env: Record<string, string>
    args: string[]
    type: string
    command: string
  }
  active: boolean
  user: string
  createdAt: string
  updatedAt: string
}

export interface CreateMcpRequest {
  name: string
  config: Record<string, unknown>
  active: boolean
}

export interface UpdateMcpRequest extends CreateMcpRequest {
  id?: string
}

// ---- Query: MCP list ----

export function useMcpList() {
  return useQuery({
    key: ['mcp'],
    query: async () => {
      const { data } = await client.get({
        url: '/mcp/',
        throwOnError: true,
      }) as { data: { items: MCPListItem[] } }
      return data.items
    },
  })
}

// ---- Mutations ----

export function useCreateOrUpdateMcp() {
  const queryCache = useQueryCache()
  return useMutation({
    mutation: (data: UpdateMcpRequest) => {
      const isEdit = !!data.id
      return isEdit
        ? client.put({ url: `/mcp/${data.id}`, body: data, throwOnError: true })
        : client.post({ url: '/mcp/', body: data, throwOnError: true })
    },
    onSettled: () => queryCache.invalidateQueries({ key: ['mcp'] }),
  })
}

export function useDeleteMcp() {
  const queryCache = useQueryCache()
  return useMutation({
    mutation: (id: string) =>
      client.delete({ url: `/mcp/${id}`, throwOnError: true }),
    onSettled: () => queryCache.invalidateQueries({ key: ['mcp'] }),
  })
}
