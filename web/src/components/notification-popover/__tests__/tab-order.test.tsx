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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { expect, test } from 'vitest'

import { useNotifications } from '@/hooks/use-notifications'
import { STATUS_QUERY_KEY } from '@/lib/status-query'

import { NotificationPopover } from '../../notification-popover'

function NotificationCenter() {
  const notifications = useNotifications()

  return (
    <NotificationPopover
      open={notifications.popoverOpen}
      onOpenChange={notifications.setPopoverOpen}
      unreadCount={notifications.unreadCount}
      activeTab={notifications.activeTab}
      onTabChange={notifications.setActiveTab}
      notice={notifications.notice}
      announcements={notifications.announcements}
      loading={notifications.loading}
    />
  )
}

function renderNotificationCenter() {
  const client = new QueryClient()
  client.setQueryData(STATUS_QUERY_KEY, { announcements_enabled: false })
  client.setQueryData(['notice'], { success: true, data: '' })

  render(
    <QueryClientProvider client={client}>
      <NotificationCenter />
    </QueryClientProvider>
  )
}

test('opening the popover selects Timeline and places it before Notice', async () => {
  const user = userEvent.setup()
  renderNotificationCenter()
  await user.click(screen.getByRole('button', { name: 'Notifications' }))

  const tabList = await screen.findByRole('tablist')
  const timeline = within(tabList).getByRole('tab', {
    name: 'Timeline',
    selected: true,
  })
  const notice = within(tabList).getByRole('tab', { name: 'Notice' })

  expect(within(tabList).getAllByRole('tab')).toEqual([timeline, notice])
})

test('reopening the popover selects Timeline after viewing Notice', async () => {
  const user = userEvent.setup()
  renderNotificationCenter()
  const trigger = screen.getByRole('button', { name: 'Notifications' })

  await user.click(trigger)
  await user.click(await screen.findByRole('tab', { name: 'Notice' }))
  expect(
    screen.getByRole('tab', { name: 'Notice', selected: true })
  ).toBeVisible()
  await user.click(screen.getByRole('button', { name: 'Close' }))
  await user.click(trigger)

  expect(
    await screen.findByRole('tab', { name: 'Timeline', selected: true })
  ).toBeVisible()
})

test('Home moves from Notice to Timeline and Enter opens it', async () => {
  const user = userEvent.setup()
  renderNotificationCenter()
  await user.click(screen.getByRole('button', { name: 'Notifications' }))

  const notice = await screen.findByRole('tab', { name: 'Notice' })
  await user.click(notice)
  await user.keyboard('{Home}{Enter}')

  const timeline = screen.getByRole('tab', { name: 'Timeline', selected: true })
  expect(timeline).toHaveFocus()
  expect(screen.getByRole('tabpanel', { name: 'Timeline' })).toHaveTextContent(
    'No system announcements'
  )
})
