import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Main } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'

import { getTransferRecords } from './api'
import { getDefaultTransferRecordDateRange } from './lib/date-range'

const PAGE_SIZE = 20

export function TransferRecords() {
  const { t } = useTranslation()
  const [draftFilters, setDraftFilters] = useState(() => ({
    username: '',
    ...getDefaultTransferRecordDateRange(),
  }))
  const [filters, setFilters] = useState(draftFilters)
  const [page, setPage] = useState(1)
  const query = useQuery({
    queryKey: ['transfer-records', page, filters],
    queryFn: () =>
      getTransferRecords({
        page,
        pageSize: PAGE_SIZE,
        username: filters.username || undefined,
        startTimestamp: filters.start
          ? Math.floor(new Date(filters.start).getTime() / 1000)
          : undefined,
        endTimestamp: filters.end
          ? Math.floor(new Date(filters.end).getTime() / 1000)
          : undefined,
      }),
  })
  const totalPages = Math.max(
    1,
    Math.ceil((query.data?.total || 0) / PAGE_SIZE)
  )

  const applyFilters = () => {
    setPage(1)
    setFilters({ ...draftFilters, username: draftFilters.username.trim() })
  }

  return (
    <Main>
      <div className='flex min-h-0 flex-1 flex-col gap-4 overflow-auto p-4 sm:p-6'>
        <div>
          <h1 className='text-xl font-semibold'>{t('Transfer Records')}</h1>
          <p className='text-muted-foreground text-sm'>
            {t('Manual quota addition history')}
          </p>
        </div>
        <FieldGroup className='flex-row flex-wrap items-end gap-2'>
          <Field className='w-auto gap-1'>
            <FieldLabel htmlFor='transfer-record-username'>
              {t('Username')}
            </FieldLabel>
            <Input
              id='transfer-record-username'
              className='w-44'
              value={draftFilters.username}
              onChange={(e) =>
                setDraftFilters((current) => ({
                  ...current,
                  username: e.target.value,
                }))
              }
              placeholder={t('Search username')}
            />
          </Field>
          <Field className='w-auto gap-1'>
            <FieldLabel htmlFor='transfer-record-start-time'>
              {t('Start time')}
            </FieldLabel>
            <Input
              id='transfer-record-start-time'
              className='w-48'
              type='datetime-local'
              value={draftFilters.start}
              onChange={(e) =>
                setDraftFilters((current) => ({
                  ...current,
                  start: e.target.value,
                }))
              }
            />
          </Field>
          <Field className='w-auto gap-1'>
            <FieldLabel htmlFor='transfer-record-end-time'>
              {t('End time')}
            </FieldLabel>
            <Input
              id='transfer-record-end-time'
              className='w-48'
              type='datetime-local'
              value={draftFilters.end}
              onChange={(e) =>
                setDraftFilters((current) => ({
                  ...current,
                  end: e.target.value,
                }))
              }
            />
          </Field>
          <Button onClick={applyFilters}>{t('Search')}</Button>
        </FieldGroup>
        <div className='overflow-hidden rounded-lg border'>
          <table className='w-full text-sm'>
            <thead className='bg-muted/50'>
              <tr className='border-b text-left'>
                <th className='px-4 py-3'>{t('Time')}</th>
                <th className='px-4 py-3'>{t('Username')}</th>
                <th className='px-4 py-3 text-right'>{t('Quota')}</th>
              </tr>
            </thead>
            <tbody>
              {query.data?.items.map((record) => (
                <tr key={record.id} className='border-b last:border-0'>
                  <td className='px-4 py-3'>
                    {new Date(record.created_at * 1000).toLocaleString()}
                  </td>
                  <td className='px-4 py-3'>{record.username}</td>
                  <td className='px-4 py-3 text-right font-mono'>
                    {record.amount}
                  </td>
                </tr>
              ))}
              {!query.isLoading && !query.data?.items.length && (
                <tr>
                  <td
                    colSpan={3}
                    className='text-muted-foreground px-4 py-10 text-center'
                  >
                    {t('No transfer records')}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
        <div className='flex items-center justify-end gap-2'>
          <Button
            variant='outline'
            size='sm'
            disabled={page <= 1}
            onClick={() => setPage((p) => p - 1)}
          >
            {t('Previous')}
          </Button>
          <span className='text-muted-foreground text-sm'>
            {page} / {totalPages}
          </span>
          <Button
            variant='outline'
            size='sm'
            disabled={page >= totalPages}
            onClick={() => setPage((p) => p + 1)}
          >
            {t('Next')}
          </Button>
        </div>
      </div>
    </Main>
  )
}
