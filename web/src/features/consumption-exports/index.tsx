/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import {
  Add01Icon,
  ArrowLeft01Icon,
  ArrowRight01Icon,
  Delete02Icon,
  RefreshIcon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Main } from '@/components/layout'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

import {
  cancelConsumptionExport,
  createConsumptionExport,
  deleteConsumptionExport,
  downloadConsumptionExport,
  listConsumptionExports,
} from './api'
import { CreateExportDialog } from './components/create-export-dialog'
import { ExportTaskTable } from './components/export-task-table'
import type {
  ConsumptionExportStatusFilter,
  ConsumptionExportTask,
  CreateConsumptionExportInput,
} from './types'

const PAGE_SIZE = 10
const ACTIVE_POLL_INTERVAL_MS = 10_000

const statusOptions: Array<{
  value: ConsumptionExportStatusFilter
  label: string
}> = [
  { value: 'all', label: 'All' },
  { value: 'pending', label: 'Pending' },
  { value: 'running', label: 'Running' },
  { value: 'succeeded', label: 'Completed' },
  { value: 'failed', label: 'Failed' },
  { value: 'canceled', label: 'Cancelled' },
]

export function ConsumptionExports() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [status, setStatus] = useState<ConsumptionExportStatusFilter>('all')
  const [createOpen, setCreateOpen] = useState(false)
  const [deleteTask, setDeleteTask] = useState<ConsumptionExportTask | null>(
    null
  )
  const [downloadingTaskId, setDownloadingTaskId] = useState<string | null>(
    null
  )

  const tasksQuery = useQuery({
    queryKey: ['consumption-exports', page, status],
    queryFn: async () => {
      const response = await listConsumptionExports({
        page,
        pageSize: PAGE_SIZE,
        status,
      })
      if (!response.success || !response.data) {
        throw new Error(
          response.message || t('Could not load consumption exports.')
        )
      }
      return response.data
    },
    staleTime: ACTIVE_POLL_INTERVAL_MS,
    refetchInterval: (query) => {
      const tasks = query.state.data?.items ?? []
      if (
        tasks.some(
          (task) => task.status === 'pending' || task.status === 'running'
        )
      ) {
        return ACTIVE_POLL_INTERVAL_MS
      }
      const now = Date.now() / 1000
      const availableExpirations = tasks
        .filter((task) => task.available && task.result.expires_at > now)
        .map((task) => task.result.expires_at)
      if (availableExpirations.length === 0) return false
      const nextExpiration = Math.min(...availableExpirations)
      return Math.min(60_000, Math.max(1_000, (nextExpiration - now) * 1000))
    },
  })

  const createMutation = useMutation({
    mutationFn: createConsumptionExport,
  })
  const cancelMutation = useMutation({
    mutationFn: cancelConsumptionExport,
  })
  const deleteMutation = useMutation({
    mutationFn: deleteConsumptionExport,
  })

  const invalidateTasks = async () => {
    await queryClient.invalidateQueries({ queryKey: ['consumption-exports'] })
  }

  const handleCreate = async (
    input: CreateConsumptionExportInput
  ): Promise<boolean> => {
    try {
      const response = await createMutation.mutateAsync(input)
      if (!response.success) return false
      toast.success(
        response.data?.created
          ? t('Export task created')
          : t('An export task is already running')
      )
      setPage(1)
      setStatus('all')
      await invalidateTasks()
      return true
    } catch {
      return false
    }
  }

  const handleCancel = async (task: ConsumptionExportTask) => {
    try {
      const response = await cancelMutation.mutateAsync(task.task_id)
      if (!response.success) return
      toast.success(t('Export canceled'))
      await invalidateTasks()
    } catch {
      return
    }
  }

  const handleDelete = async () => {
    if (!deleteTask) return
    try {
      const response = await deleteMutation.mutateAsync(deleteTask.task_id)
      if (!response.success) return
      toast.success(t('Export task deleted'))
      setDeleteTask(null)
      await invalidateTasks()
    } catch {
      return
    }
  }

  const handleDownload = async (task: ConsumptionExportTask) => {
    setDownloadingTaskId(task.task_id)
    try {
      const blob = await downloadConsumptionExport(task.task_id)
      const objectURL = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = objectURL
      link.download = task.result.file_name || 'consumption.xlsx'
      document.body.appendChild(link)
      link.click()
      link.remove()
      URL.revokeObjectURL(objectURL)
    } catch {
      toast.error(t('Download failed'))
      await invalidateTasks()
    } finally {
      setDownloadingTaskId(null)
    }
  }

  const tasks = tasksQuery.data?.items ?? []
  const total = tasksQuery.data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  const hasActiveTasks = tasks.some(
    (task) => task.status === 'pending' || task.status === 'running'
  )
  const selectedStatusLabel =
    statusOptions.find((option) => option.value === status)?.label ?? 'All'

  return (
    <Main>
      <div className='min-h-0 flex-1 overflow-auto px-3 py-3 sm:px-4 sm:py-6'>
        <section className='bg-card mx-auto w-full max-w-[1600px] overflow-hidden rounded-lg border shadow-xs'>
          <header className='flex flex-col gap-4 border-b px-4 py-4 sm:flex-row sm:items-start sm:justify-between sm:px-5'>
            <div className='min-w-0'>
              <div className='flex flex-wrap items-center gap-2'>
                <h1 className='text-base font-semibold'>
                  {t('Consumption exports')}
                </h1>
                <Badge variant='secondary'>{total}</Badge>
              </div>
              <p className='text-muted-foreground mt-1 text-sm'>
                {t(
                  'Export files remain available for one hour and are then removed automatically.'
                )}
              </p>
            </div>
            <div className='flex w-full flex-wrap items-center gap-2 sm:w-auto sm:flex-nowrap'>
              {hasActiveTasks && (
                <span
                  className='text-muted-foreground mr-auto text-xs tabular-nums sm:mr-0'
                  aria-live='polite'
                >
                  {t('Refreshing every {{seconds}}s', {
                    seconds: ACTIVE_POLL_INTERVAL_MS / 1000,
                  })}
                </span>
              )}
              <Tooltip>
                <TooltipTrigger
                  render={
                    <Button
                      type='button'
                      variant='outline'
                      size='icon'
                      aria-label={t('Refresh')}
                      disabled={tasksQuery.isFetching}
                      onClick={() => void tasksQuery.refetch()}
                    />
                  }
                >
                  <HugeiconsIcon icon={RefreshIcon} strokeWidth={2} />
                </TooltipTrigger>
                <TooltipContent>{t('Refresh')}</TooltipContent>
              </Tooltip>
              <Button type='button' onClick={() => setCreateOpen(true)}>
                <HugeiconsIcon
                  icon={Add01Icon}
                  strokeWidth={2}
                  data-icon='inline-start'
                />
                {t('Create export')}
              </Button>
            </div>
          </header>

          <div className='flex items-center justify-between gap-3 border-b px-4 py-3 sm:px-5'>
            <Select
              items={statusOptions}
              value={status}
              onValueChange={(value) => {
                setStatus(value ?? 'all')
                setPage(1)
              }}
            >
              <SelectTrigger>
                <SelectValue>{t(selectedStatusLabel)}</SelectValue>
              </SelectTrigger>
              <SelectContent alignItemWithTrigger={false}>
                <SelectGroup>
                  {statusOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {t(option.label)}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>

          <ExportTaskTable
            tasks={tasks}
            loading={tasksQuery.isLoading}
            downloadingTaskId={downloadingTaskId}
            cancelingTaskId={
              cancelMutation.isPending ? cancelMutation.variables : null
            }
            onCancel={(task) => void handleCancel(task)}
            onDelete={setDeleteTask}
            onDownload={(task) => void handleDownload(task)}
          />

          <footer className='flex items-center justify-between gap-4 border-t px-4 py-3 sm:px-5'>
            <span className='text-muted-foreground text-sm'>
              {t('{{start}}-{{end}} of {{total}}', {
                start: total === 0 ? 0 : (page - 1) * PAGE_SIZE + 1,
                end: Math.min(page * PAGE_SIZE, total),
                total,
              })}
            </span>
            <div className='flex items-center gap-2'>
              <Button
                type='button'
                variant='outline'
                size='icon-sm'
                aria-label={t('Previous page')}
                disabled={page <= 1}
                onClick={() => setPage((current) => Math.max(1, current - 1))}
              >
                <HugeiconsIcon icon={ArrowLeft01Icon} strokeWidth={2} />
              </Button>
              <span className='min-w-8 text-center text-sm tabular-nums'>
                {page}
              </span>
              <Button
                type='button'
                variant='outline'
                size='icon-sm'
                aria-label={t('Next page')}
                disabled={page >= totalPages}
                onClick={() =>
                  setPage((current) => Math.min(totalPages, current + 1))
                }
              >
                <HugeiconsIcon icon={ArrowRight01Icon} strokeWidth={2} />
              </Button>
            </div>
          </footer>
        </section>
      </div>

      <CreateExportDialog
        open={createOpen}
        pending={createMutation.isPending}
        onOpenChange={setCreateOpen}
        onCreate={handleCreate}
      />

      <AlertDialog
        open={deleteTask !== null}
        onOpenChange={(open) => {
          if (!open) setDeleteTask(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogMedia>
              <HugeiconsIcon icon={Delete02Icon} strokeWidth={2} />
            </AlertDialogMedia>
            <AlertDialogTitle>{t('Delete export task?')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('The generated file and task history will be removed.')}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('Cancel')}</AlertDialogCancel>
            <AlertDialogAction
              variant='destructive'
              disabled={deleteMutation.isPending}
              onClick={() => void handleDelete()}
            >
              {t('Delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Main>
  )
}
