import { boolean, jsonb, pgTable, text, uuid } from 'drizzle-orm/pg-core'
import { users } from './users'

export const mcpConnection = pgTable('mcp_connection', {
  id: uuid('id').primaryKey().defaultRandom(),
  type: text('type').notNull(),
  name: text('name').notNull(),
  config: jsonb('config').notNull(),
  active: boolean('active').notNull().default(true),
  user: uuid('user').notNull().references(() => users.id),
})