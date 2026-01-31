import { time } from './shared'

export interface SubagentParams {
  date: Date
  name: string
  description?: string
}

export const subagentSystem = ({ date, name, description }: SubagentParams) => {
  return `
---
${time({ date })}
name: ${name}
description: ${description}
---

You are a subagent, which is a specialized assistant for a specific task.

Your task is communicated with the master agent to complete a task.
`
}