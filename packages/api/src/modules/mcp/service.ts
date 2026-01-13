import { db } from '@memoh/db'
import { mcpConnection } from '@memoh/db/schema'
import { eq, desc, asc, sql } from 'drizzle-orm'
import { calculateOffset, createPaginatedResult, type PaginatedResult } from '../../utils/pagination'
import type { CreateMCPConnectionInput, UpdateMCPConnectionInput } from './model'

/**
 * MCP Connection 列表返回类型
 */
type MCPConnectionListItem = {
  id: string
  type: string
  name: string
  config: unknown
  active: boolean
  user: string
  createdAt: Date
  updatedAt: Date
}

/**
 * 获取用户的所有 MCP 连接（支持分页）
 */
export const getMCPConnections = async (
  userId: string,
  params?: {
    limit?: number
    page?: number
    sortOrder?: 'asc' | 'desc'
  }
): Promise<PaginatedResult<MCPConnectionListItem>> => {
  const limit = params?.limit || 10
  const page = params?.page || 1
  const sortOrder = params?.sortOrder || 'desc'
  const offset = calculateOffset(page, limit)

  // 获取总数
  const [{ count }] = await db
    .select({ count: sql<number>`count(*)` })
    .from(mcpConnection)
    .where(eq(mcpConnection.user, userId))

  // 获取分页数据
  const orderFn = sortOrder === 'desc' ? desc : asc
  const connections = await db
    .select()
    .from(mcpConnection)
    .where(eq(mcpConnection.user, userId))
    .orderBy(orderFn(mcpConnection.id))
    .limit(limit)
    .offset(offset)

  // 类型转换
  const formattedConnections = connections.map(conn => ({
    id: conn.id,
    type: conn.type,
    name: conn.name,
    config: conn.config,
    active: conn.active,
    user: conn.user,
    createdAt: new Date(),
    updatedAt: new Date(),
  }))

  return createPaginatedResult(formattedConnections, Number(count), page, limit)
}

/**
 * 获取用户的所有活跃 MCP 连接
 */
export const getActiveMCPConnections = async (
  userId: string
) => {
  const connections = await db
    .select()
    .from(mcpConnection)
    .where(eq(mcpConnection.user, userId))
    .orderBy(desc(mcpConnection.id))

  return connections.filter(conn => conn.active).map(conn => ({
    id: conn.id,
    type: conn.type,
    name: conn.name,
    config: conn.config,
    active: conn.active,
    user: conn.user,
  }))
}

/**
 * 根据 ID 获取单个 MCP 连接
 */
export const getMCPConnection = async (
  connectionId: string
) => {
  const [result] = await db
    .select()
    .from(mcpConnection)
    .where(eq(mcpConnection.id, connectionId))
  
  if (!result) {
    return null
  }

  return {
    id: result.id,
    type: result.type,
    name: result.name,
    config: result.config,
    active: result.active,
    user: result.user,
  }
}

/**
 * 创建新的 MCP 连接
 */
export const createMCPConnection = async (
  userId: string,
  data: CreateMCPConnectionInput
) => {
  const [newConnection] = await db
    .insert(mcpConnection)
    .values({
      user: userId,
      type: data.config.type,
      name: data.name,
      config: data.config,
      active: data.active,
    })
    .returning()

  return {
    id: newConnection.id,
    type: newConnection.type,
    name: newConnection.name,
    config: newConnection.config,
    active: newConnection.active,
    user: newConnection.user,
  }
}

/**
 * 更新 MCP 连接
 */
export const updateMCPConnection = async (
  connectionId: string,
  userId: string,
  data: UpdateMCPConnectionInput
) => {
  // 检查 MCP 连接是否存在且属于该用户
  const existingConnection = await getMCPConnection(connectionId)
  if (!existingConnection) {
    return null
  }
  
  if (existingConnection.user !== userId) {
    throw new Error('Forbidden: You do not have permission to update this MCP connection')
  }

  const updateData: {
    name?: string
    config?: unknown
    type?: string
    active?: boolean
  } = {}
  
  if (data.name !== undefined) {
    updateData.name = data.name
  }
  if (data.config !== undefined) {
    updateData.config = data.config
    updateData.type = data.config.type
  }
  if (data.active !== undefined) {
    updateData.active = data.active
  }

  const [updatedConnection] = await db
    .update(mcpConnection)
    .set(updateData)
    .where(eq(mcpConnection.id, connectionId))
    .returning()
  
  return {
    id: updatedConnection.id,
    type: updatedConnection.type,
    name: updatedConnection.name,
    config: updatedConnection.config,
    active: updatedConnection.active,
    user: updatedConnection.user,
  }
}

/**
 * 删除 MCP 连接
 */
export const deleteMCPConnection = async (
  connectionId: string,
  userId: string
) => {
  // 检查 MCP 连接是否存在且属于该用户
  const existingConnection = await getMCPConnection(connectionId)
  if (!existingConnection) {
    return null
  }
  
  if (existingConnection.user !== userId) {
    throw new Error('Forbidden: You do not have permission to delete this MCP connection')
  }

  const [deletedConnection] = await db
    .delete(mcpConnection)
    .where(eq(mcpConnection.id, connectionId))
    .returning()
  
  return {
    id: deletedConnection.id,
    type: deletedConnection.type,
    name: deletedConnection.name,
    config: deletedConnection.config,
    active: deletedConnection.active,
    user: deletedConnection.user,
  }
}

/**
 * 设置 MCP 连接的活跃状态
 */
export const setMCPConnectionActive = async (
  connectionId: string,
  active: boolean
) => {
  await db
    .update(mcpConnection)
    .set({ active })
    .where(eq(mcpConnection.id, connectionId))
}

