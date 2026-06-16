import { useEffect, useState } from 'react'
import { api } from '../services/api'
import type { Entity } from '../services/types'
import { Page } from './Dashboard'

const titles: Record<Entity['kind'], string> = {
  prompt: 'Prompts',
  skill: 'Skills',
  agent: 'Agents',
  plugin: 'Plugins',
}

const descriptions: Record<Entity['kind'], string> = {
  prompt: 'Search, install, update, and uninstall reusable prompts.',
  skill: 'Search, install, update, and uninstall Claude-style skills.',
  agent: 'Search, install, update, and uninstall custom agents.',
  plugin: 'Search, install, update, and uninstall assistant plugins.',
}

type LibraryProps = {
  kind: Entity['kind']
}

export function Library({ kind }: LibraryProps) {
  const [query, setQuery] = useState('')
  const [items, setItems] = useState<Entity[]>([])

  useEffect(() => { void api.listEntities(kind).then(setItems) }, [kind])
  async function search() { setItems(await api.searchEntities(kind, query)) }

  return <Page title={titles[kind]} description={descriptions[kind]}>
    <div className="inline-form"><input aria-label={`${titles[kind]} search`} value={query} onChange={(event) => setQuery(event.target.value)} placeholder={`Search ${titles[kind].toLowerCase()}`} /><button onClick={search}>Search</button></div>
    <div className="cards">{items.map((item) => <article className="card" key={`${item.kind}-${item.name}`}><h2>{item.name}</h2><p>{item.description}</p><small>{item.apps?.join(', ')}</small></article>)}</div>
  </Page>
}
