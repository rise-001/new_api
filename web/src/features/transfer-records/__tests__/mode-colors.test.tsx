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
import { render, screen } from '@testing-library/react'
import { expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import { TransferRecords } from '../index'

test('transfer record types use distinct semantic colors', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      success: true,
      data: {
        total: 5,
        items: ['topup', 'add', 'subtract', 'override', 'redemption'].map(
          (mode, id) => ({
            id,
            user_id: id + 1,
            username: `user-${id + 1}`,
            created_at: 1,
            amount: '100',
            mode,
          })
        ),
      },
    },
  })

  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={queryClient}>
      <TransferRecords />
    </QueryClientProvider>
  )
  await screen.findByText('user-1')

  const expectedColors = [
    ['Recharge', 'text-info'],
    ['Add', 'text-success'],
    ['Subtract', 'text-destructive'],
    ['Override', 'text-warning'],
    ['Redemption Code Redemption', 'text-chart-4'],
  ]

  for (const [label, color] of expectedColors) {
    const badges = await screen.findAllByText(label)
    const badge = badges
      .map((element) => element.closest('[data-slot="status-badge"]'))
      .find((element) => element !== null)
    expect(badge).toHaveClass(color)
  }
})
