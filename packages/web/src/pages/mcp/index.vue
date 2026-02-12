<template>
  <section class="[&_td:last-child]:w-40">
    <CreateMCP />
    <DataTable
      :columns="columns"
      :data="mcpFormatData"
    />
  </section>
</template>

<script setup lang="ts">
import { h, provide, ref, computed } from 'vue'
import DataTable from '@/components/data-table/index.vue'
import CreateMCP from '@/components/create-mcp/index.vue'
import { type ColumnDef } from '@tanstack/vue-table'
import {
  Badge,
  Button
} from '@memoh/ui'
import { type MCPListItem as MCPType } from '@/composables/api/useMcp'
import { useMcpList, useDeleteMcp } from '@/composables/api/useMcp'
import { useI18n } from 'vue-i18n'

const open = ref(false)
const editMCPData = ref<{
  name: string
  config: MCPType['config']
  active: boolean
  id: string
} | null>(null)
provide('open', open)
provide('mcpEditData', editMCPData)

const { mutate: DeleteMCP } = useDeleteMcp()
const { t } = useI18n()

const columns:ColumnDef<MCPType>[] = [
  {
    accessorKey: 'name',
    header: () => h('div', { class: 'text-left py-4' }, t('mcp.table.name')),
   
  },
  {
    accessorKey: 'type',
    header: () => h('div', { class: 'text-left' }, t('mcp.table.type')),
  },
  {
    accessorKey: 'config.command',
    header: () => h('div', { class: 'text-left' }, t('mcp.table.command')),
  },
  {
    accessorKey: 'config.cwd',
    header: () => h('div', { class: 'text-left' }, t('mcp.table.cwd')),
  },
  {
    accessorKey: 'config.args',
    header: () => h('div', { class: 'text-left' }, t('mcp.table.arguments')),
    cell: ({ row }) => h('div', {class:'flex gap-4'}, row.original.config.args.map((argTxt) => {
      return h(Badge, {
        variant:'default'
      },()=>argTxt)
    }))
  },
  {
    accessorKey: 'config.env',
    header: () => h('div', { class: 'text-left' }, t('mcp.table.env')),
    cell: ({ row }) => h('div', { class: 'flex gap-4' }, Object.entries(row.original.config.env).map(([key,value]) => {
      return h(Badge, {
        variant: 'outline'
      }, ()=>`${key}:${value}`)
    }))
  },
  {
    accessorKey: 'control',
    header: () => h('div', { class: 'text-center' }, t('common.operation')),
    cell: ({ row }) => h('div', {class:'flex gap-2'}, [
      h(Button, {
        onClick() {
          editMCPData.value = {
            name: row.original.name,
            config: {...row.original.config},
            active: row.original.active,
            id:row.original.id
          }       
          open.value=true
        }
      }, ()=>t('common.edit')),
      h(Button, {
        variant: 'destructive',
        async onClick() {        
          try {
            await DeleteMCP(row.original.id)
          } catch {
            return
          }
        }
      },()=>t('common.delete'))
    ])
  }
]

const { data: mcpData } = useMcpList()

const mcpFormatData = computed(() => mcpData.value ?? [])

</script>