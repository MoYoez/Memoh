import { Schedule } from '../types'
import { stringify } from 'yaml'

export interface ScheduleParams {
  schedule: Schedule
  date: Date
}

export const schedule = (params: ScheduleParams) => {
  const headers = {
    'schedule-name': params.schedule.name,
    'schedule-description': params.schedule.description,
    'max-calls': params.schedule.maxCalls ?? 'Unlimited',
    'cron-pattern': params.schedule.pattern,
  }
  return `
** This is a scheduled task automatically send to you by the system **
---
${stringify(headers)}
---

${params.schedule.command}
  `.trim()
}