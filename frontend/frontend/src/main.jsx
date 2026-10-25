import React, { useCallback, useEffect, useMemo, useState } from 'react'
import { createRoot } from 'react-dom/client'
import './style.css'

const API = import.meta.env.VITE_API_URL || 'http://localhost:8080'
const prettyTime = value => value ? new Date(value).toLocaleString('ru-RU', { day:'2-digit', month:'short', hour:'2-digit', minute:'2-digit', second:'2-digit' }) : 'Ещё не проверялся'

function App() {
  const [services, setServices] = useState([])
  const [name, setName] = useState('')
  const [url, setUrl] = useState('')
  const [filter, setFilter] = useState('all')
  const [busy, setBusy] = useState({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const load = useCallback(async () => {
    try {
      const response = await fetch(`${API}/api/services`)
      if (!response.ok) throw new Error('API unavailable')
      setServices(await response.json())
      setError('')
    } catch { setError('Не удаётся подключиться к backend. Проверьте VITE_API_URL и доступность API.') }
    finally { setLoading(false) }
  }, [])

  useEffect(() => { load() }, [load])

  async function addService(e) {
    e.preventDefault()
    try {
      const response = await fetch(`${API}/api/services`, { method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({name,url}) })
      const data = await response.json()
      if (!response.ok) throw new Error(data.error || 'Ошибка создания')
      setServices(current => [data, ...current]); setName(''); setUrl(''); setError('')
    } catch (err) { setError(err.message || 'Не удалось добавить сервис') }
  }

  async function checkService(id) {
    setBusy(current => ({...current, [id]:true}))
    try {
      const response = await fetch(`${API}/api/services/${id}/check`, { method:'POST' })
      const data = await response.json()
      if (!response.ok) throw new Error(data.error || 'Ошибка проверки')
      setServices(current => current.map(s => s.id === id ? data : s))
    } catch (err) { setError(err.message || 'Не удалось проверить сервис') }
    finally { setBusy(current => ({...current, [id]:false})) }
  }

  async function deleteService(id) {
    try {
      const response = await fetch(`${API}/api/services/${id}`, { method:'DELETE' })
      if (!response.ok) throw new Error('Не удалось удалить сервис')
      setServices(current => current.filter(s => s.id !== id))
    } catch (err) { setError(err.message) }
  }

  const filtered = useMemo(() => services.filter(s => filter === 'all' || s.status === filter), [services, filter])
  const up = services.filter(s => s.status === 'up').length
  const down = services.filter(s => s.status === 'down').length

  return <div className="app">
    <aside className="sidebar">
      <div className="logo"><span className="logo-icon">ϟ</span><span>pulse<span className="logo-dot">.</span></span></div>
      <div className="side-label">WORKSPACE</div>
      <div className="nav-item selected"><span>▦</span> Обзор сервисов <b>●</b></div>
      <div className="side-bottom"><div className="avatar">OP</div><div><strong>Оператор</strong><small>Инфраструктура</small></div><span className="dots">···</span></div>
    </aside>
    <main className="main">
      <header><div className="crumb">Мониторинг <span>/</span> <strong>Обзор</strong></div><div className="live"><i/> SYSTEM CONSOLE</div></header>
      <section className="intro"><div><div className="kicker">INFRASTRUCTURE / HEALTH</div><h1>Состояние сервисов<span>.</span></h1><p>Все ключевые endpoints — в одном месте. Проверяйте доступность и следите за откликом.</p></div><button className="refresh" onClick={load}>↻ <span>Обновить</span></button></section>
      <section className="metrics">
        <article className="metric"><div className="metric-top"><span>Подключено сервисов</span><span className="metric-icon">▦</span></div><strong>{services.length.toString().padStart(2,'0')}</strong><small>Всего endpoints</small></article>
        <article className="metric"><div className="metric-top"><span>В норме</span><span className="status-dot green"/> </div><strong>{up.toString().padStart(2,'0')}</strong><small className="good">● Работают штатно</small></article>
        <article className="metric"><div className="metric-top"><span>Недоступны</span><span className="status-dot red"/></div><strong>{down.toString().padStart(2,'0')}</strong><small className={down ? 'bad' : ''}>{down ? '● Требуют внимания' : 'Нет активных инцидентов'}</small></article>
        <article className="metric accent"><div className="metric-top"><span>Последняя проверка</span><span className="metric-icon">◷</span></div><strong className="last">{services.some(s=>s.last_checked_at) ? prettyTime([...services].filter(s=>s.last_checked_at).sort((a,b)=>new Date(b.last_checked_at)-new Date(a.last_checked_at))[0].last_checked_at) : '—'}</strong><small>По всем сервисам</small></article>
      </section>
      <section className="panel">
        <div className="panel-heading"><div><div className="kicker">LIVE INVENTORY</div><h2>Мониторинг endpoints</h2></div><span className="pill"><i/> {services.length} сервисов</span></div>
        <form className="add-form" onSubmit={addService}><label><span>НАЗВАНИЕ</span><input required value={name} onChange={e=>setName(e.target.value)} placeholder="Например, Public API"/></label><label className="url-field"><span>URL ДЛЯ ПРОВЕРКИ</span><input required type="url" value={url} onChange={e=>setUrl(e.target.value)} placeholder="https://example.com/health"/></label><button className="add-button" type="submit">＋ Добавить</button></form>
        {error && <div className="error">{error}<button onClick={load}>Повторить</button></div>}
        <div className="table-head"><div><div className="kicker">REGISTERED SERVICES</div><h3>Список сервисов</h3></div><div className="filters">{[['all','Все'],['up','Работают'],['down','Проблемы']].map(([key,label])=><button key={key} className={filter===key?'active':''} onClick={()=>setFilter(key)}>{label}</button>)}</div></div>
        <div className="service-list">
          {loading ? <div className="empty">Загружаем данные из PostgreSQL…</div> : filtered.length===0 ? <div className="empty"><span>⌁</span><strong>Сервисы не найдены</strong><p>Добавьте endpoint выше и запустите проверку.</p></div> : filtered.map((s,index)=><article className="service-row" key={s.id}>
            <div className={`service-symbol ${s.status}`} >{s.name.slice(0,1).toUpperCase()}</div>
            <div className="service-info"><strong>{s.name}</strong><a href={s.url} target="_blank" rel="noreferrer">{s.url}</a><small>Проверка: {prettyTime(s.last_checked_at)}</small></div>
            <div className="response">{s.response_ms===null ? '—' : `${s.response_ms} ms`}<small>{s.status_code ? `HTTP ${s.status_code}` : 'Нет данных'}</small></div>
            <span className={`state ${s.status}`}><i/>{s.status==='up'?'В норме':s.status==='down'?'Недоступен':'Не проверен'}</span>
            <button className="check-btn" disabled={busy[s.id]} onClick={()=>checkService(s.id)}>{busy[s.id]?'…':'↻ Проверить'}</button>
            <button className="remove" title="Удалить сервис" onClick={()=>deleteService(s.id)}>×</button>
          </article>)}
        </div>
        <footer><span><i/> Данные хранятся в PostgreSQL</span><span>Обновлено вручную · <button onClick={load}>Перезагрузить</button></span></footer>
      </section>
      <div className="bottom-note"><span>ϟ</span> PULSE MONITOR <i/> BUILT FOR RELIABLE RELEASES <b>v1.0.0</b></div>
    </main>
  </div>
}
createRoot(document.getElementById('root')).render(<App/>)
