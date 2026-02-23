import { stringify } from 'yaml'

export interface SubagentParams {
  date: Date
  name: string
  description?: string
}

export const subagentSystem = ({ date, name, description }: SubagentParams) => {
  const headers = {
    'name': name,
    'description': description,
    'time-now': date.toISOString(),
  }
  return [
    description,
    '---'
    + stringify(headers)
    + '---'
  ].join('\n\n')
}