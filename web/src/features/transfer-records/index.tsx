import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Main } from '@/components/layout'
import { StatusBadge, type StatusVariant } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'

import { getTransferRecords, type TransferRecordMode } from './api'
import { TransferUserFilter } from './components/user-filter'
import { getDefaultTransferRecordDateRange } from './lib/date-range'

function ModeBadge(props: { mode: string }) {
  const { t } = useTranslation()
  let label = '—'
  let variant: StatusVariant = 'neutral'
  if (props.mode === 'add') {
    label = t('Add')
    variant = 'success'
  } else if (props.mode === 'subtract') {
    label = t('Subtract')
    variant = 'danger'
  } else if (props.mode === 'override') {
    label = t('Override')
    variant = 'info'
  } else if (props.mode === 'topup') {
    label = t('Recharge')
    variant = 'success'
  } else if (props.mode === 'redemption') {
    label = t('Redemption Code Redemption')
    variant = 'purple'
  }
  return <StatusBadge label={label} variant={variant} copyable={false} />
}

export function TransferRecords() {
  const { t } = useTranslation()
  const [draftFilters, setDraftFilters] = useState(() => ({
    username: '',
    mode: '' as TransferRecordMode | '',
    pageSize: 20,
    ...getDefaultTransferRecordDateRange(),
  }))
  const [filters, setFilters] = useState(draftFilters)
  const [page, setPage] = useState(1)
  const query = useQuery({
    queryKey: ['transfer-records', page, filters],
    queryFn: () =>
      getTransferRecords({
        page,
        pageSize: filters.pageSize,
        mode: filters.mode,
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
    Math.ceil((query.data?.total || 0) / filters.pageSize)
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
            {t('Recharge, quota adjustment and redemption history')}
          </p>
        </div>
        <form
          onSubmit={(event) => {
            event.preventDefault()
            applyFilters()
          }}
        >
          <FieldGroup className='flex-row flex-wrap items-end gap-2'>
            <Field className='w-auto gap-1'>
              <FieldLabel htmlFor='transfer-record-username'>
                {t('Username')}
              </FieldLabel>
              <TransferUserFilter
                value={draftFilters.username}
                onValueChange={(username) =>
                  setDraftFilters((current) => ({
                    ...current,
                    username,
                  }))
                }
              />
            </Field>
            <Field className='w-auto gap-1'>
              <FieldLabel htmlFor='transfer-record-mode'>
                {t('Type')}
              </FieldLabel>
              <NativeSelect
                id='transfer-record-mode'
                className='w-44'
                value={draftFilters.mode}
                onChange={(event) =>
                  setDraftFilters((current) => ({
                    ...current,
                    mode: event.target.value as TransferRecordMode | '',
                  }))
                }
              >
                <NativeSelectOption value=''>
                  {t('All Types')}
                </NativeSelectOption>
                <NativeSelectOption value='topup'>
                  {t('Recharge')}
                </NativeSelectOption>
                <NativeSelectOption value='add'>{t('Add')}</NativeSelectOption>
                <NativeSelectOption value='subtract'>
                  {t('Subtract')}
                </NativeSelectOption>
                <NativeSelectOption value='override'>
                  {t('Override')}
                </NativeSelectOption>
                <NativeSelectOption value='redemption'>
                  {t('Redemption Code Redemption')}
                </NativeSelectOption>
              </NativeSelect>
            </Field>
            <Field className='w-auto gap-1'>
              <FieldLabel htmlFor='transfer-record-page-size'>
                {t('Rows per page')}
              </FieldLabel>
              <NativeSelect
                id='transfer-record-page-size'
                className='w-24'
                value={draftFilters.pageSize}
                onChange={(event) =>
                  setDraftFilters((current) => ({
                    ...current,
                    pageSize: Number(event.target.value),
                  }))
                }
              >
                {[20, 50, 100].map((size) => (
                  <NativeSelectOption key={size} value={size}>
                    {size}
                  </NativeSelectOption>
                ))}
              </NativeSelect>
            </Field>
            <Field className='w-auto gap-1'>
              <FieldLabel htmlFor='transfer-record-start-time'>
                {t('Start Time')}
              </FieldLabel>
              <Input
                id='transfer-record-start-time'
                className='w-48'
                type='datetime-local'
                max={draftFilters.end || undefined}
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
                {t('End Time')}
              </FieldLabel>
              <Input
                id='transfer-record-end-time'
                className='w-48'
                type='datetime-local'
                min={draftFilters.start || undefined}
                value={draftFilters.end}
                onChange={(e) =>
                  setDraftFilters((current) => ({
                    ...current,
                    end: e.target.value,
                  }))
                }
              />
            </Field>
            <Button type='submit' disabled={query.isFetching}>
              {t('Search')}
            </Button>
          </FieldGroup>
        </form>
        <div className='overflow-x-auto rounded-lg border'>
          <table className='w-full text-sm' aria-busy={query.isFetching}>
            <thead className='bg-muted/50'>
              <tr className='border-b text-left'>
                <th className='px-4 py-3'>{t('Time')}</th>
                <th className='px-4 py-3'>{t('Username')}</th>
                <th className='px-4 py-3'>{t('Type')}</th>
                <th className='px-4 py-3 text-right'>{t('Quota')}</th>
              </tr>
            </thead>
            <tbody>
              {query.data?.items.map((record) => (
                <tr key={record.id} className='border-b last:border-0'>
                  <td className='px-4 py-3'>
                    {new Date(record.created_at * 1000).toLocaleString()}
                  </td>
                  <td className='px-4 py-3'>
                    {record.username || `#${record.user_id}`}
                  </td>
                  <td className='px-4 py-3'>
                    <ModeBadge mode={record.mode} />
                  </td>
                  <td className='px-4 py-3 text-right font-mono'>
                    {record.amount}
                  </td>
                </tr>
              ))}
              {query.isLoading && (
                <tr>
                  <td
                    colSpan={4}
                    className='text-muted-foreground px-4 py-10 text-center'
                  >
                    {t('Loading...')}
                  </td>
                </tr>
              )}
              {query.isError && (
                <tr>
                  <td
                    colSpan={4}
                    className='text-destructive px-4 py-10 text-center'
                    role='alert'
                  >
                    {t('Failed to load transfer records')}
                  </td>
                </tr>
              )}
              {!query.isLoading &&
                !query.isError &&
                !query.data?.items.length && (
                  <tr>
                    <td
                      colSpan={4}
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
            disabled={query.isFetching || page <= 1}
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
            disabled={query.isFetching || page >= totalPages}
            onClick={() => setPage((p) => p + 1)}
          >
            {t('Next')}
          </Button>
        </div>
      </div>
    </Main>
  )
}
