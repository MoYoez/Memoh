import { z } from 'zod'

// 创建 Schedule 的 Schema
const CreateScheduleSchema = z.object({
  name: z.string().min(1, 'Name is required').max(100),
  description: z.string().min(1, 'Description is required'),
  command: z.string().min(1, 'Command is required'),
  pattern: z.string().min(1, 'Cron pattern is required'),
  maxCalls: z.number().int().positive().optional(),
})

// 更新 Schedule 的 Schema
const UpdateScheduleSchema = z.object({
  name: z.string().min(1).max(100).optional(),
  description: z.string().optional(),
  command: z.string().optional(),
  pattern: z.string().optional(),
  maxCalls: z.number().int().positive().optional(),
  active: z.boolean().optional(),
})

// 查询参数 Schema
const GetSchedulesQuerySchema = z.object({
  page: z.string().optional(),
  limit: z.string().optional(),
  sortOrder: z.enum(['asc', 'desc']).optional(),
})

export type CreateScheduleInput = z.infer<typeof CreateScheduleSchema>
export type UpdateScheduleInput = z.infer<typeof UpdateScheduleSchema>
export type GetSchedulesQuery = z.infer<typeof GetSchedulesQuerySchema>

export const CreateScheduleModel = {
  body: CreateScheduleSchema,
}

export const UpdateScheduleModel = {
  params: z.object({
    id: z.string().uuid('Invalid schedule ID format'),
  }),
  body: UpdateScheduleSchema,
}

export const GetScheduleByIdModel = {
  params: z.object({
    id: z.string().uuid('Invalid schedule ID format'),
  }),
}

export const DeleteScheduleModel = {
  params: z.object({
    id: z.string().uuid('Invalid schedule ID format'),
  }),
}

export const GetSchedulesModel = {
  query: GetSchedulesQuerySchema,
}

