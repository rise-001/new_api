import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Main } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { formatQuota } from '@/lib/format'

import { getTransferRecords } from './api'

const PAGE_SIZE = 20

export function TransferRecords() {
  const { t } = useTranslation()
  const [username, setUsername] = useState('')
  const [start, setStart] = useState('')
  const [end, setEnd] = useState('')
  const [filters, setFilters] = useState({ username: '', start: '', end: '' })
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
  const totalPages = Math.max(1, Math.ceil((query.data?.total || 0) / PAGE_SIZE))

  const applyFilters = () => {
    setPage(1)
    setFilters({ username: username.trim(), start, end })
  }

  return (
    <Main>
      <div className='flex min-h-0 flex-1 flex-col gap-4 overflow-auto p-4 sm:p-6'>
        <div>
          <h1 className='text-xl font-semibold'>{t('Transfer Records')}</h1>
          <p className='text-muted-foreground text-sm'>{t('Affiliate quota transfer history')}</p>
        </div>
        <div className='flex flex-wrap items-end gap-2'>
          <label className='grid gap-1 text-sm'>
            {t('Username')}
            <Input
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder={t('Search username')}
            />
          </label>
          <label className='grid gap-1 text-sm'>
            {t('Start time')}
            <Input type='datetime-local' value={start} onChange={(e) => setStart(e.target.value)} />
          </label>
          <label className='grid gap-1 text-sm'>
            {t('End time')}
            <Input type='datetime-local' value={end} onChange={(e) => setEnd(e.target.value)} />
          </label>
          <Button onClick={applyFilters}>{t('Search')}</Button>
        </div>
        <div className='overflow-hidden rounded-lg border'>
          <table className='w-full text-sm'>
            <thead className='bg-muted/50'>
              <tr className='border-b text-left'>
                <th className='px-4 py-3'>{t('Time')}</th>
                <th className='px-4 py-3'>{t('Username')}</th>
                <th className='px-4 py-3 text-right'>{t('Transfer Amount')}</th>
              </tr>
            </thead>
            <tbody>
              {query.data?.items.map((record) => (
                <tr key={record.id} className='border-b last:border-0'>
                  <td className='px-4 py-3'>{new Date(record.created_at * 1000).toLocaleString()}</td>
                  <td className='px-4 py-3'>{record.username}</td>
                  <td className='px-4 py-3 text-right font-mono'>{formatQuota(record.quota)}</td>
                </tr>
              ))}
              {!query.isLoading && !query.data?.items.length && (
                <tr><td colSpan={3} className='text-muted-foreground px-4 py-10 text-center'>{t('No transfer records')}</td></tr>
              )}
            </tbody>
          </table>
        </div>
        <div className='flex items-center justify-end gap-2'>
          <Button variant='outline' size='sm' disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>{t('Previous')}</Button>
          <span className='text-muted-foreground text-sm'>{page} / {totalPages}</span>
          <Button variant='outline' size='sm' disabled={page >= totalPages} onClick={() => setPage((p) => p + 1)}>{t('Next')}</Button>
        </div>
      </div>
    </Main>
  )
}
