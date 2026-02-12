export interface robot{
  description: string
  time: Date,
  id: string | number,
  type: string,
  action: 'robot',
  state:'thinking'|'generate'|'complete',
  platform?: string
}

export interface user{
  description: string, 
  time: Date, 
  id: number | string,
  action:'user',
  platform?: string,
  senderDisplayName?: string,
  senderAvatarUrl?: string,
  isSelf?: boolean
}