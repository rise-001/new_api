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
  Delete02Icon,
  Download04Icon,
  FileExportIcon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Progress } from '@/components/ui/progress'
import { Spinner } from '@/components/ui/spinner'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import dayjs from '@/lib/dayjs'

import type { ConsumptionExportStatus, ConsumptionExportTask } from '../types'

type ExportTaskTableProps = {
  tasks: ConsumptionExportTask[]
  loading: boolean
  downloadingTaskId: string | null
  cancelingTaskId: string | null
  onCancel: (task: ConsumptionExportTask) => void
  onDelete: (task: ConsumptionExportTask) => void
  onDownload: (task: ConsumptionExportTask) => void
}

const statusBadgeVariant: Record<
  ConsumptionExportStatus,
  'secondary' | 'warning' | 'default' | 'destructive' | 'outline'
> = {
  pending: 'secondary',
  running: 'warning',
  succeeded: 'default',
  failed: 'destructive',
  canceled: 'outline',
}

const statusLabel: Record<ConsumptionExportStatus, string> = {
  pending: 'Pending',
  running: 'Running',
  succeeded: 'Completed',
  failed: 'Failed',
  canceled: 'Cancelled',
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function formatExpiry(expiresAt: number, expiredLabel: string): string {
  const remaining = expiresAt - Math.floor(Date.now() / 1000)
  if (remaining <= 0) return expiredLabel
  const minutes = Math.floor(remaining / 60)
  const seconds = remaining % 60
  return `${minutes}m ${seconds}s`
}

export function ExportTaskTable(props: ExportTaskTableProps) {
  const { t } = useTranslation()

  if (props.loading) {
    return (
      <div className='flex min-h-56 items-center justify-center' role='status'>
        <Spinner />
        <span className='sr-only'>{t('Loading')}</span>
      </div>
    )
  }

  if (props.tasks.length === 0) {
    return (
      <Empty className='min-h-64'>
        <EmptyHeader>
          <EmptyMedia variant='icon'>
            <HugeiconsIcon icon={FileExportIcon} strokeWidth={2} />
          </EmptyMedia>
          <EmptyTitle>{t('No consumption exports')}</EmptyTitle>
          <EmptyDescription>
            {t(
              'Create an export to download detailed consumption records and totals.'
            )}
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    )
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t('ID')}</TableHead>
          <TableHead>{t('Time range')}</TableHead>
          <TableHead>{t('API key')}</TableHead>
          <TableHead>{t('Status')}</TableHead>
          <TableHead>{t('Records / size')}</TableHead>
          <TableHead>{t('Expires in')}</TableHead>
          <TableHead>{t('Created at')}</TableHead>
          <TableHead className='text-right'>{t('Actions')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {props.tasks.map((task) => {
          const active = task.status === 'pending' || task.status === 'running'
          const available =
            task.available && task.result.expires_at > Date.now() / 1000
          return (
            <TableRow key={task.task_id}>
              <TableCell className='font-medium'>{task.id}</TableCell>
              <TableCell>
                <div>
                  {dayjs
                    .unix(task.payload.start_timestamp)
                    .format('YYYY-MM-DD HH:mm')}
                </div>
                <div className='text-muted-foreground'>
                  {dayjs
                    .unix(task.payload.end_timestamp)
                    .format('YYYY-MM-DD HH:mm')}
                </div>
              </TableCell>
              <TableCell>
                {task.payload.token_name || t('All API keys')}
              </TableCell>
              <TableCell className='min-w-40'>
                <div className='flex flex-col gap-2'>
                  <Badge variant={statusBadgeVariant[task.status]}>
                    {t(statusLabel[task.status])}
                  </Badge>
                  {active && (
                    <div className='flex flex-col gap-1'>
                      <Progress value={task.state.progress} />
                      <div className='text-muted-foreground text-xs'>
                        {task.state.progress}% · {task.state.processed}/
                        {task.state.total}
                      </div>
                    </div>
                  )}
                  {task.status === 'failed' && task.error && (
                    <div
                      className='text-destructive max-w-56 truncate text-xs'
                      title={task.error}
                    >
                      {task.error}
                    </div>
                  )}
                </div>
              </TableCell>
              <TableCell>
                <div>
                  {task.result.record_count || task.state.total} {t('records')}
                </div>
                {task.result.file_size > 0 && (
                  <div className='text-muted-foreground'>
                    {formatFileSize(task.result.file_size)}
                  </div>
                )}
              </TableCell>
              <TableCell>
                {task.result.expires_at > 0
                  ? formatExpiry(task.result.expires_at, t('Expired'))
                  : '-'}
              </TableCell>
              <TableCell>
                {dayjs.unix(task.created_at).format('YYYY-MM-DD HH:mm:ss')}
              </TableCell>
              <TableCell>
                <div className='flex justify-end gap-2'>
                  {active && (
                    <Button
                      type='button'
                      variant='outline'
                      size='sm'
                      disabled={props.cancelingTaskId === task.task_id}
                      onClick={() => props.onCancel(task)}
                    >
                      {props.cancelingTaskId === task.task_id && (
                        <Spinner data-icon='inline-start' />
                      )}
                      {t('Cancel export')}
                    </Button>
                  )}
                  {available && (
                    <Button
                      type='button'
                      size='sm'
                      disabled={props.downloadingTaskId === task.task_id}
                      onClick={() => props.onDownload(task)}
                    >
                      {props.downloadingTaskId === task.task_id ? (
                        <Spinner data-icon='inline-start' />
                      ) : (
                        <HugeiconsIcon
                          icon={Download04Icon}
                          strokeWidth={2}
                          data-icon='inline-start'
                        />
                      )}
                      {t('Download')}
                    </Button>
                  )}
                  {!active && (
                    <Tooltip>
                      <TooltipTrigger
                        render={
                          <Button
                            type='button'
                            variant='ghost'
                            size='icon-sm'
                            aria-label={t('Delete')}
                            onClick={() => props.onDelete(task)}
                          />
                        }
                      >
                        <HugeiconsIcon icon={Delete02Icon} strokeWidth={2} />
                      </TooltipTrigger>
                      <TooltipContent>{t('Delete')}</TooltipContent>
                    </Tooltip>
                  )}
                </div>
              </TableCell>
            </TableRow>
          )
        })}
      </TableBody>
    </Table>
  )
}
