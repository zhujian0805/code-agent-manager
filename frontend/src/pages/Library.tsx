import { useEffect, useState } from 'react'
import { api } from '../services/api'
import type { Entity } from '../services/types'
import { Page } from './Dashboard'

const kinds: Entity['kind'][] = ['prompt', 'skill', 'agent', 'plugin']

export function Library() {
  const [kind, setKind] = useState<Entity['kind']>('skill')
  const [query, setQuery] = useState('')
  const [items, setItems] = useState<Entity[]>([])

  useEffect(() => { void api.listEntities(kind).then(setItems) }, [kind])
  async function search() { setItems(await api.searchEntities(kind, query)) }

  return <Page title="Library" description="Search, install, update, and uninstall prompts, skills, agents, and plugins.">
    <div className="actions">{kinds.map((item) => <button key={item} className={kind === item ? 'primary' : ''} onClick={() => setKind(item)}>{item}s</button>)}</div>
    <div className="inline-form"><input aria-label="Library search" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search local library" /><button onClick={search}>Search</button></div>
    <div className="cards">{items.map((item) => <article className="card" key={`${item.kind}-${item.name}`}><h2>{item.name}</h2><p>{item.description}</p><small>{item.apps?.join(', ')}</small></article>)}</div>
  </Page>
}
