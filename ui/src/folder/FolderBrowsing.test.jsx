import React from 'react'
import { render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import FolderList from './FolderList'
import FolderShow from './FolderShow'

const { nothing } = vi.hoisted(() => ({ nothing: () => null }))
const Records = React.createContext([])
const folders = [
  { id: 'old-root', library_id: '1', parent_id: '', missing: true },
  { id: 'root', library_id: '1', parent_id: '', missing: false },
  { id: 'musics', parent_id: 'root', name: 'Musics', missing: true },
  { id: 'classical', parent_id: 'root', name: 'Classical', missing: false },
  {
    id: 'empty-parent',
    parent_id: 'root',
    name: 'Empty parent',
    missing: false,
    num_audio_files: 0,
  },
  { id: 'old-child', parent_id: 'classical', name: 'Old child', missing: true },
  {
    id: 'child',
    parent_id: 'classical',
    name: 'Current child',
    missing: false,
  },
  {
    id: 'nested-empty',
    parent_id: 'classical',
    name: 'Nested empty parent',
    missing: false,
    num_audio_files: 0,
  },
]
const matching = (filter) =>
  folders.filter((folder) =>
    Object.entries(filter).every(([key, value]) => folder[key] === value),
  )
const getList = vi.fn((resource, { filter, pagination }) =>
  Promise.resolve({
    data:
      resource === 'library'
        ? [{ id: 1 }]
        : matching(filter).slice(0, pagination.perPage),
  }),
)
const dataProvider = { getList }
const referenceFilters = vi.fn()
const record = { id: 'classical', name: 'Classical' }

vi.mock('react-redux', () => ({ useSelector: () => ({ grid: true }) }))
vi.mock('./FolderListActions', () => ({ default: () => null }))
vi.mock('./FolderSongs', () => ({ default: () => null }))
vi.mock('./Breadcrumbs', () => ({ default: () => null }))
vi.mock('../common', () => ({
  useResourceRefresh: () => {},
  Title: () => null,
  FolderContextMenu: () => null,
  DurationField: () => null,
  SizeField: () => null,
}))
vi.mock('./FolderGridView', () => ({
  default: function FolderGridViewMock() {
    const records = React.useContext(Records)
    return <div>{records.map((folder) => folder.name).join(', ')}</div>
  },
}))
vi.mock('react-admin', () => ({
  useTranslate: () => (value) => value,
  useDataProvider: () => dataProvider,
  useShowController: () => ({ record, loading: false }),
  useShowContext: () => ({ record, loading: false }),
  useListContext: () => ({
    total: React.useContext(Records).length,
    loaded: true,
    filterValues: {},
  }),
  List: ({ filter, children }) => (
    <Records.Provider value={matching(filter)}>{children}</Records.Provider>
  ),
  ReferenceManyField: ({ reference, target, filter = {}, children }) => {
    if (reference !== 'folder') return null
    referenceFilters({ target, filter })
    return (
      <Records.Provider value={matching({ [target]: record.id, ...filter })}>
        {children}
      </Records.Provider>
    )
  },
  ShowContextProvider: ({ children }) => children,
  SimpleShowLayout: ({ children }) => children,
  Datagrid: nothing,
  FunctionField: nothing,
  Pagination: nothing,
  NumberField: nothing,
  Filter: nothing,
  SearchInput: nothing,
  ListToolbar: nothing,
  Title: nothing,
}))

describe('Folder browsing', () => {
  beforeEach(() => vi.clearAllMocks())

  it('selects a live root and hides missing root children without hiding empty parents', async () => {
    render(<FolderList />)
    await waitFor(() =>
      expect(screen.getByText(/Classical/)).toBeInTheDocument(),
    )
    expect(getList).toHaveBeenCalledWith(
      'folder',
      expect.objectContaining({
        filter: { library_id: '1', parent_id: '', missing: false },
      }),
    )
    expect(screen.queryByText(/Musics/)).not.toBeInTheDocument()
    expect(screen.getByText(/Empty parent/)).toBeInTheDocument()
  })

  it('hides missing nested folders while retaining live empty parents', () => {
    render(<FolderShow />)
    expect(referenceFilters).toHaveBeenCalledWith({
      target: 'parent_id',
      filter: { missing: false },
    })
    expect(screen.queryByText(/Old child/)).not.toBeInTheDocument()
    expect(screen.getByText(/Current child/)).toBeInTheDocument()
    expect(screen.getByText(/Nested empty parent/)).toBeInTheDocument()
  })
})
