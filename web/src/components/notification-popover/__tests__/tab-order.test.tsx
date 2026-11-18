import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
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
import { useState } from 'react'
import { expect, test } from 'vitest'

import { NotificationPopover } from '../../notification-popover'

function OpenNotificationPopover() {
  const [activeTab, setActiveTab] = useState<'notice' | 'announcements'>(
    'notice'
  )

  return (
    <NotificationPopover
      open
      onOpenChange={() => {}}
      unreadCount={0}
      activeTab={activeTab}
      onTabChange={setActiveTab}
      notice=''
      announcements={[]}
      loading={false}
    />
  )
}

test('the open popover places Timeline before the selected Notice tab', async () => {
  render(<OpenNotificationPopover />)

  const tabList = await screen.findByRole('tablist')
  const timeline = within(tabList).getByRole('tab', { name: 'Timeline' })
  const notice = within(tabList).getByRole('tab', {
    name: 'Notice',
    selected: true,
  })

  expect(within(tabList).getAllByRole('tab')).toEqual([timeline, notice])
})

test('Home moves from Notice to Timeline and Enter opens it', async () => {
  const user = userEvent.setup()
  render(<OpenNotificationPopover />)

  const notice = await screen.findByRole('tab', { name: 'Notice' })
  await user.click(notice)
  await user.keyboard('{Home}{Enter}')

  const timeline = screen.getByRole('tab', { name: 'Timeline', selected: true })
  expect(timeline).toHaveFocus()
  expect(screen.getByRole('tabpanel', { name: 'Timeline' })).toHaveTextContent(
    'No system announcements'
  )
})
