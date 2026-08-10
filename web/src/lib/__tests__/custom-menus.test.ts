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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  CUSTOM_MENU_LOCATION,
  CUSTOM_MENU_OPEN_MODE,
  parseCustomMenuItems,
  serializeCustomMenuItems,
} from '../custom-menus.ts'

describe('custom menu configuration', () => {
  test('parses valid chat and top navigation entries', () => {
    const items = parseCustomMenuItems(
      JSON.stringify([
        {
          id: 'support',
          url: 'https://support.example.com',
          name: 'Support',
          location: 'chat',
          open_mode: 'embed',
        },
        {
          id: 'docs',
          url: 'https://docs.example.com',
          name: 'Docs',
          location: 'top',
          open_mode: 'new_tab',
        },
      ])
    )

    assert.equal(items.length, 2)
    assert.equal(items[0].openMode, CUSTOM_MENU_OPEN_MODE.EMBED)
    assert.equal(items[1].location, CUSTOM_MENU_LOCATION.TOP)
  })

  test('drops unsafe URLs and embedded top navigation entries', () => {
    const items = parseCustomMenuItems([
      {
        id: 'unsafe',
        url: 'javascript:alert(1)',
        name: 'Unsafe',
        location: 'chat',
        open_mode: 'embed',
      },
      {
        id: 'embedded-top',
        url: 'https://docs.example.com',
        name: 'Docs',
        location: 'top',
        open_mode: 'embed',
      },
    ])

    assert.deepEqual(items, [])
  })

  test('serializes top navigation entries as new-tab links', () => {
    const serialized = serializeCustomMenuItems([
      {
        id: 'docs',
        url: 'https://docs.example.com',
        name: 'Docs',
        location: CUSTOM_MENU_LOCATION.TOP,
        openMode: CUSTOM_MENU_OPEN_MODE.EMBED,
      },
    ])

    assert.deepEqual(JSON.parse(serialized), [
      {
        id: 'docs',
        url: 'https://docs.example.com',
        name: 'Docs',
        location: 'top',
        open_mode: 'new_tab',
      },
    ])
  })
})
