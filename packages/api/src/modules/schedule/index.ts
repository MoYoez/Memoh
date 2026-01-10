import Elysia from 'elysia'
import { authMiddleware } from '../../middlewares/auth'
import {
  CreateScheduleModel,
  UpdateScheduleModel,
  GetScheduleByIdModel,
  DeleteScheduleModel,
  GetSchedulesModel,
} from './model'
import {
  getSchedules,
  getSchedule,
  createSchedule,
  updateSchedule,
  deleteSchedule,
  createScheduler,
} from './service'

export const { scheduleTask, resume } = createScheduler()

export const scheduleModule = new Elysia({ prefix: '/schedule' })
  .use(authMiddleware)
  // Get all schedules for current user
  .onStart(async () => {
    await resume()
  })
  .get('/', async ({ user, query }) => {
    try {
      const page = parseInt(query.page as string) || 1
      const limit = parseInt(query.limit as string) || 10
      const sortOrder = (query.sortOrder as string) || 'desc'

      const result = await getSchedules(user.userId, {
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
        error: error instanceof Error ? error.message : 'Failed to fetch schedules',
      }
    }
  }, GetSchedulesModel)
  // Get schedule by ID
  .get('/:id', async ({ user, params, set }) => {
    try {
      const schedule = await getSchedule(params.id)
      
      if (!schedule) {
        set.status = 404
        return {
          success: false,
          error: 'Schedule not found',
        }
      }

      if (schedule.user !== user.userId) {
        set.status = 403
        return {
          success: false,
          error: 'Forbidden: You do not have permission to access this schedule',
        }
      }

      return {
        success: true,
        data: schedule,
      }
    } catch (error) {
      set.status = 500
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Failed to fetch schedule',
      }
    }
  }, GetScheduleByIdModel)
  // Create new schedule
  .post('/', async ({ user, body, set }) => {
    try {
      const newSchedule = await createSchedule(user.userId, body)
      
      // 启动定时任务
      scheduleTask(user.userId, {
        id: newSchedule.id!,
        pattern: newSchedule.pattern,
        name: newSchedule.name,
        description: newSchedule.description,
        command: newSchedule.command,
        maxCalls: newSchedule.maxCalls || undefined,
      })

      set.status = 201
      return {
        success: true,
        data: newSchedule,
      }
    } catch (error) {
      set.status = 500
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Failed to create schedule',
      }
    }
  }, CreateScheduleModel)
  // Update schedule
  .put('/:id', async ({ user, params, body, set }) => {
    try {
      const updatedSchedule = await updateSchedule(params.id, user.userId, body)
      
      if (!updatedSchedule) {
        set.status = 404
        return {
          success: false,
          error: 'Schedule not found',
        }
      }

      return {
        success: true,
        data: updatedSchedule,
      }
    } catch (error) {
      if (error instanceof Error && error.message.includes('Forbidden')) {
        set.status = 403
      } else {
        set.status = 500
      }
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Failed to update schedule',
      }
    }
  }, UpdateScheduleModel)
  // Delete schedule
  .delete('/:id', async ({ user, params, set }) => {
    try {
      const deletedSchedule = await deleteSchedule(params.id, user.userId)
      
      if (!deletedSchedule) {
        set.status = 404
        return {
          success: false,
          error: 'Schedule not found',
        }
      }

      return {
        success: true,
        data: deletedSchedule,
      }
    } catch (error) {
      if (error instanceof Error && error.message.includes('Forbidden')) {
        set.status = 403
      } else {
        set.status = 500
      }
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Failed to delete schedule',
      }
    }
  }, DeleteScheduleModel)
