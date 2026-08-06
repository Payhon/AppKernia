import { create } from 'zustand'

export type SidebarMode = 'expanded' | 'collapsed' | 'hidden'

export interface SidebarStorage {
  getItem: (key: string) => string | null
  setItem: (key: string, value: string) => void
}

interface SidebarState {
  mode: SidebarMode
  setMode: (mode: SidebarMode) => void
}

export const sidebarModeStorageKey = 'ak.admin.sidebar-mode.v1'

function browserStorage(): SidebarStorage | undefined {
  try {
    return typeof window === 'undefined' ? undefined : window.localStorage
  } catch {
    return undefined
  }
}

export function readSidebarMode(storage: SidebarStorage | null | undefined = browserStorage()): SidebarMode {
  try {
    const value = storage?.getItem(sidebarModeStorageKey)
    return value === 'collapsed' || value === 'hidden' || value === 'expanded' ? value : 'expanded'
  } catch {
    return 'expanded'
  }
}

export function writeSidebarMode(mode: SidebarMode, storage: SidebarStorage | null | undefined = browserStorage()) {
  try {
    storage?.setItem(sidebarModeStorageKey, mode)
  } catch {
    // A blocked or full localStorage must not make navigation unusable.
  }
}

export const useSidebarStore = create<SidebarState>((set) => ({
  mode: readSidebarMode(),
  setMode: (mode) => {
    writeSidebarMode(mode)
    set({ mode })
  },
}))
