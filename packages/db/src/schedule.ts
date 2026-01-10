import { boolean, integer, pgTable, text, timestamp, uuid } from 'drizzle-orm/pg-core'
import { users } from './users'

export const schedule = pgTable('schedule', {
  id: uuid('id').primaryKey().defaultRandom(),
  name: text('name').notNull(),
  description: text('description').notNull(),
  command: text('command').notNull(),
  pattern: text('pattern').notNull(),
  maxCalls: integer('max_calls'),
  user: uuid('user').notNull().references(() => users.id),
  createdAt: timestamp('created_at').notNull().defaultNow(),
  updatedAt: timestamp('updated_at').notNull().defaultNow(),
  active: boolean('active').notNull().default(true),
})