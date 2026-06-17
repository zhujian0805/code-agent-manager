import { ReactNode } from 'react'

// Page is the shared content frame for every routed view: a max-width column
// with a title/description header. Extracted from the former Dashboard module so
// no page has to import another page just to reuse the layout shell.
export function Page({ title, description, children }: { title: string; description: string; children: ReactNode }) {
  return <main className="page"><header><h1>{title}</h1><p>{description}</p></header>{children}</main>
}
