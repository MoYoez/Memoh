import Elysia from 'elysia'
import { authMiddleware } from '../../middlewares/auth'
import {
  CreateMCPConnectionModel,
  UpdateMCPConnectionModel,
  GetMCPConnectionByIdModel,
  DeleteMCPConnectionModel,
  GetMCPConnectionsModel,
} from './model'
import {
  getMCPConnections,
  getMCPConnection,
  createMCPConnection,
  updateMCPConnection,
  deleteMCPConnection,
} from './service'

export const mcpModule = new Elysia({ prefix: '/mcp' })
  .use(authMiddleware)
  // Get all MCP connections for current user
  .get('/', async ({ user, query }) => {
    try {
      const page = parseInt(query.page as string) || 1
      const limit = parseInt(query.limit as string) || 10
      const sortOrder = (query.sortOrder as string) || 'desc'

      const result = await getMCPConnections(user.userId, {
        page,
        limit,
        sortOrder: sortOrder as 'asc' | 'desc',
      })

      return {
        success: true,
        ...result,
      }
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Failed to fetch MCP connections',
      }
    }
  }, GetMCPConnectionsModel)
  // Get MCP connection by ID
  .get('/:id', async ({ user, params, set }) => {
    try {
      const connection = await getMCPConnection(params.id)
      
      if (!connection) {
        set.status = 404
        return {
          success: false,
          error: 'MCP connection not found',
        }
      }

      if (connection.user !== user.userId) {
        set.status = 403
        return {
          success: false,
          error: 'Forbidden: You do not have permission to access this MCP connection',
        }
      }

      return {
        success: true,
        data: connection,
      }
    } catch (error) {
      set.status = 500
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Failed to fetch MCP connection',
      }
    }
  }, GetMCPConnectionByIdModel)
  // Create new MCP connection
  .post('/', async ({ user, body, set }) => {
    try {
      const newConnection = await createMCPConnection(user.userId, body)

      set.status = 201
      return {
        success: true,
        data: newConnection,
      }
    } catch (error) {
      set.status = 500
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Failed to create MCP connection',
      }
    }
  }, CreateMCPConnectionModel)
  // Update MCP connection
  .put('/:id', async ({ user, params, body, set }) => {
    try {
      const updatedConnection = await updateMCPConnection(params.id, user.userId, body)
      
      if (!updatedConnection) {
        set.status = 404
        return {
          success: false,
          error: 'MCP connection not found',
        }
      }

      return {
        success: true,
        data: updatedConnection,
      }
    } catch (error) {
      if (error instanceof Error && error.message.includes('Forbidden')) {
        set.status = 403
      } else {
        set.status = 500
      }
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Failed to update MCP connection',
      }
    }
  }, UpdateMCPConnectionModel)
  // Delete MCP connection
  .delete('/:id', async ({ user, params, set }) => {
    try {
      const deletedConnection = await deleteMCPConnection(params.id, user.userId)
      
      if (!deletedConnection) {
        set.status = 404
        return {
          success: false,
          error: 'MCP connection not found',
        }
      }

      return {
        success: true,
        data: deletedConnection,
      }
    } catch (error) {
      if (error instanceof Error && error.message.includes('Forbidden')) {
        set.status = 403
      } else {
        set.status = 500
      }
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Failed to delete MCP connection',
      }
    }
  }, DeleteMCPConnectionModel)
