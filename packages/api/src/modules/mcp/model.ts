import { z } from 'zod'

// Stdio MCP 连接配置
const StdioMCPConnectionSchema = z.object({
  type: z.literal('stdio'),
  name: z.string().min(1, 'Name is required').max(100),
  active: z.boolean(),
  command: z.string().min(1, 'Command is required'),
  args: z.array(z.string()),
  env: z.record(z.string(), z.string()),
  cwd: z.string(),
})

// HTTP MCP 连接配置
const HTTPMCPConnectionSchema = z.object({
  type: z.literal('http'),
  name: z.string().min(1, 'Name is required').max(100),
  active: z.boolean(),
  url: z.string().url('Invalid URL'),
  headers: z.record(z.string(), z.string()),
})

// SSE MCP 连接配置
const SSEMCPConnectionSchema = z.object({
  type: z.literal('sse'),
  name: z.string().min(1, 'Name is required').max(100),
  active: z.boolean(),
  url: z.string().url('Invalid URL'),
  headers: z.record(z.string(), z.string()),
})

// 联合类型
const MCPConnectionConfigSchema = z.union([
  StdioMCPConnectionSchema,
  HTTPMCPConnectionSchema,
  SSEMCPConnectionSchema,
])

// 创建 MCP 连接的 Schema
const CreateMCPConnectionSchema = z.object({
  name: z.string().min(1, 'Name is required').max(100),
  config: MCPConnectionConfigSchema,
  active: z.boolean().default(true),
})

// 更新 MCP 连接的 Schema
const UpdateMCPConnectionSchema = z.object({
  name: z.string().min(1).max(100).optional(),
  config: MCPConnectionConfigSchema.optional(),
  active: z.boolean().optional(),
})

// 查询参数 Schema
const GetMCPConnectionsQuerySchema = z.object({
  page: z.string().optional(),
  limit: z.string().optional(),
  sortOrder: z.enum(['asc', 'desc']).optional(),
})

export type CreateMCPConnectionInput = z.infer<typeof CreateMCPConnectionSchema>
export type UpdateMCPConnectionInput = z.infer<typeof UpdateMCPConnectionSchema>
export type GetMCPConnectionsQuery = z.infer<typeof GetMCPConnectionsQuerySchema>

export const CreateMCPConnectionModel = {
  body: CreateMCPConnectionSchema,
}

export const UpdateMCPConnectionModel = {
  params: z.object({
    id: z.string().uuid('Invalid MCP connection ID format'),
  }),
  body: UpdateMCPConnectionSchema,
}

export const GetMCPConnectionByIdModel = {
  params: z.object({
    id: z.string().uuid('Invalid MCP connection ID format'),
  }),
}

export const DeleteMCPConnectionModel = {
  params: z.object({
    id: z.string().uuid('Invalid MCP connection ID format'),
  }),
}

export const GetMCPConnectionsModel = {
  query: GetMCPConnectionsQuerySchema,
}

