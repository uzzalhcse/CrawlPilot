<script setup lang="ts" generic="TData">
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '@/components/ui/table'

interface Column {
  key: string
  label: string
  sortable?: boolean
  align?: 'left' | 'center' | 'right'
  class?: string
}

interface Props {
  data: TData[]
  columns: Column[]
  loading?: boolean
  onRowClick?: (row: TData) => void
}

defineProps<Props>()
</script>

<template>
  <div class="w-full">
    <Table>
      <TableHeader>
        <TableRow class="hover:bg-transparent border-b">
          <TableHead
            v-for="column in columns"
            :key="column.key"
            :class="[
              'h-10 px-6 text-xs font-medium text-muted-foreground',
              column.align === 'right' ? 'text-right' : column.align === 'center' ? 'text-center' : 'text-left',
              column.class
            ]"
          >
            {{ column.label }}
            <span v-if="column.sortable" class="ml-1">↑</span>
          </TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        <TableRow
          v-for="(row, index) in data"
          :key="index"
          :class="[
            'border-b cursor-pointer hover:bg-muted/50 transition-colors',
            onRowClick && 'cursor-pointer'
          ]"
          @click="onRowClick?.(row)"
        >
          <slot name="row" :row="row" :index="index" />
        </TableRow>
      </TableBody>
    </Table>
  </div>
</template>
