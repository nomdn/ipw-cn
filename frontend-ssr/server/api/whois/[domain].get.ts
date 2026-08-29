import { defineEventHandler, createError } from 'h3'

const IANA_BOOTSTRAP_URL = 'https://data.iana.org/rdap/dns.json'
// bootstrap 数据基本不变，按进程缓存 24h，避免每个请求都重新下载
const BOOTSTRAP_TTL = 24 * 60 * 60 * 1000
let bootstrapCache: { map: Map<string, string>; at: number } | null = null

function encodeDomain(input: string): string {
  try {
    const url = new URL(/^https?:\/\//i.test(input) ? input : `https://${input}`)
    return url.hostname.toLowerCase().replace(/^www\./, '')
  } catch {
    return ''
  }
}

function getTLDCandidates(hostname: string): string[] {
  const parts = hostname.toLowerCase().split('.')
  const candidates: string[] = []
  for (let i = parts.length - 1; i >= 1; i--) {
    candidates.push(parts.slice(i).join('.'))
  }
  return candidates
}

async function fetchBootstrap(): Promise<Map<string, string>> {
  if (bootstrapCache && Date.now() - bootstrapCache.at < BOOTSTRAP_TTL) {
    return bootstrapCache.map
  }
  const res = await fetch(IANA_BOOTSTRAP_URL, {
    headers: { accept: 'application/json' },
    signal: AbortSignal.timeout(10_000),
  })
  if (!res.ok) {
    throw createError({
      statusCode: 502,
      statusMessage: `Failed to fetch IANA bootstrap (HTTP ${res.status})`,
    })
  }
  const data = await res.json()
  const map = new Map<string, string>()
  for (const entry of data.services ?? []) {
    const rawTlds = entry[0]
    const rawUrls = entry[1]
    if (!Array.isArray(rawTlds) || !Array.isArray(rawUrls)) continue
    if (rawTlds.length === 0 || rawUrls.length === 0) continue
    const baseUrl = String(rawUrls[0]).replace(/\/+$/, '')
    for (const tld of rawTlds) {
      map.set(String(tld), baseUrl)
    }
  }
  bootstrapCache = { map, at: Date.now() }
  return map
}

async function findRdapBase(domain: string): Promise<string | null> {
  const bootstrap = await fetchBootstrap()
  const candidates = getTLDCandidates(domain)
  for (const tld of candidates) {
    if (bootstrap.has(tld)) {
      return bootstrap.get(tld) ?? null
    }
  }
  return null
}

function extractVcard(entity: any): Record<string, string> {
  const vcard = entity.vcardArray
  const result: Record<string, string> = {}
  if (vcard && Array.isArray(vcard[1])) {
    for (const item of vcard[1]) {
      const key = item[0]
      if (key === 'fn') result.name = String(item[3] ?? item[2] ?? '')
      else if (key === 'org') result.org = String(item[3] ?? item[2] ?? '')
      else if (key === 'tel') result.phone = String(item[3] ?? '')
      else if (key === 'email') result.email = String(item[3] ?? '')
      else if (key === 'adr') {
        const adr = item[3]
        if (Array.isArray(adr)) {
          const parts = adr.filter((s: any) => s && String(s).trim())
          if (parts.length > 4) result.province = String(parts[4] ?? '')
        }
      }
      else if (key === 'contact-uri') result.contactUri = String(item[3] ?? '')
    }
  }
  return result
}

function extractContact(entities: any[], roles: string[]): Record<string, string> {
  // Search top-level entities first
  for (const entity of entities) {
    if (!roles.some((r) => entity.roles?.includes(r))) continue
    const vcard = extractVcard(entity)
    if (Object.keys(vcard).length > 0) return vcard
  }
  // Search sub-entities (common for abuse contacts nested under registrar)
  for (const entity of entities) {
    for (const sub of entity.entities ?? []) {
      if (!roles.some((r) => sub.roles?.includes(r))) continue
      const vcard = extractVcard(sub)
      if (Object.keys(vcard).length > 0) return vcard
    }
  }
  return {}
}

function extractRegistrar(entities: any[]): { name: string; ianaId: string } | null {
  for (const entity of entities) {
    if (entity.roles?.includes('registrar')) {
      const vcard = entity.vcardArray
      let name = ''
      if (vcard && Array.isArray(vcard[1])) {
        const fn = vcard[1].find((i: any) => i[0] === 'fn')
        if (fn) name = String(fn[3] ?? fn[2] ?? '')
      }
      const ianaIdEntry = entity.publicIds?.find((p: any) => p.type === 'IANA Registrar ID')
      const ianaId = ianaIdEntry ? String(ianaIdEntry.identifier) : ''
      return { name, ianaId }
    }
  }
  return null
}

function extractDates(events: any[]): { registration: string; expiration: string; lastChanged: string } {
  const dates: Record<string, string> = {}
  for (const ev of events) {
    const action = ev.eventAction
    const date = ev.eventDate
    if (action === 'registration') dates.registration = date
    else if (action === 'expiration') dates.expiration = date
    else if (action === 'last changed') dates.lastChanged = date
  }
  return {
    registration: dates.registration ?? '',
    expiration: dates.expiration ?? '',
    lastChanged: dates.lastChanged ?? '',
  }
}

function formatRdapResponse(rdap: any, whoisServer: string): any {
  const entities = rdap.entities ?? []
  const registrar = extractRegistrar(entities)
  const abuseContact = extractContact(entities, ['abuse'])
  const registrantContact = extractContact(entities, ['registrant'])
  const technicalContact = extractContact(entities, ['technical'])

  const events = rdap.events ?? []
  const dates = extractDates(events)

  const nameservers = (rdap.nameservers ?? [])
    .map((ns: any) => ns.ldhName?.toUpperCase())
    .filter(Boolean)

  const relatedLink = (rdap.links ?? []).find((l: any) => l.rel === 'related')

  const result: any = {
    domain: (rdap.ldhName ?? rdap.unicodeName ?? '').toUpperCase(),
    status: rdap.status ?? [],
    registrar: registrar ?? { name: '', ianaId: '' },
    registrant: registrantContact,
    technical: technicalContact,
    abuseContact,
    dates,
    nameservers,
    whoisServer,
  }

  if (relatedLink?.href) {
    result.relatedRdapUrl = relatedLink.href
  }

  return result
}

function getWhoisServer(relatedHref: string): string {
  try {
    const url = new URL(relatedHref)
    const host = url.hostname.toLowerCase()
    if (host.includes('dnspod') || host.includes('tencent')) return 'whois.dnspod.cn'
    if (host.includes('verisign')) return 'whois.verisign.com'
    if (host.includes('identitydigital')) return 'whois.identitydigital.services'
    if (host.includes('publicinterestregistry')) return 'whois.pir.org'
    if (host.includes('centralnic')) return 'whois.centralnic.com'
    // Generic: strip leading "rdap." and path
    return host.replace(/^rdap\./, 'whois.')
  } catch {
    return ''
  }
}

export default defineEventHandler(async (event) => {
  const params = event.context.params
  const rawDomain = typeof params?.domain === 'string' ? params.domain : ''

  if (!rawDomain) {
    throw createError({
      statusCode: 400,
      statusMessage: 'Missing domain parameter',
    })
  }

  const domain = encodeDomain(rawDomain)

  if (!domain) {
    throw createError({
      statusCode: 400,
      statusMessage: 'Invalid domain',
    })
  }

  const rdapBase = await findRdapBase(domain)

  if (!rdapBase) {
    throw createError({
      statusCode: 404,
      statusMessage: `No RDAP server found for domain: ${domain}`,
    })
  }

  const rdapUrl = `${rdapBase}/domain/${domain}`

  const primaryRes = await fetch(rdapUrl, {
    headers: { accept: 'application/rdap+json' },
    signal: AbortSignal.timeout(15_000),
  })

  if (!primaryRes.ok) {
    throw createError({
      statusCode: 502,
      statusMessage: `RDAP server returned HTTP ${primaryRes.status}`,
    })
  }

  const primaryData = await primaryRes.json()

  // For thick registries, also fetch the registrar's RDAP for detailed data
  const relatedLink = (primaryData.links ?? []).find((l: any) => l.rel === 'related')
  let thickData: any = null

  if (relatedLink?.href) {
    try {
      const thickRes = await fetch(relatedLink.href, {
        headers: { accept: 'application/rdap+json' },
        signal: AbortSignal.timeout(15_000),
      })
      if (thickRes.ok) {
        thickData = await thickRes.json()
      }
    } catch {
      // thick registry fetch failed, continue with primary data only
    }
  }

  // Merge: always base on primary (has registrar info), overlay thick data details
  const merged = { ...primaryData }

  // Overlay entities from thick data if primary is missing registrant/abuse/technical
  if (thickData?.entities) {
    const primaryRoles = new Set((merged.entities ?? []).flatMap((e: any) => e.roles ?? []))
    const thickEntities = thickData.entities.filter((e: any) => {
      const roles = e.roles ?? []
      return roles.some((r: string) => !primaryRoles.has(r))
    })
    if (thickEntities.length) {
      merged.entities = [...(merged.entities ?? []), ...thickEntities]
    }
  }

  // Overlay nameservers from thick data if primary didn't have them
  if (thickData?.nameservers && !merged.nameservers?.length) {
    merged.nameservers = thickData.nameservers
  }

  // Overlay events from thick data if primary is missing key events
  if (thickData?.events) {
    const primaryActions = new Set((merged.events ?? []).map((e: any) => e.eventAction))
    const thickEvents = thickData.events.filter((e: any) => !primaryActions.has(e.eventAction))
    if (thickEvents.length) {
      merged.events = [...(merged.events ?? []), ...thickEvents]
    }
  }

  const whoisServer = getWhoisServer(relatedLink?.href ?? rdapBase)

  return formatRdapResponse(merged, whoisServer)
})
