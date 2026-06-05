import React, { createContext, useContext, useEffect, useState } from 'react'
import liff from '@line/liff'
import client from '../api/client.js'

const LiffContext = createContext(null)

const LIFF_ID = import.meta.env.VITE_LIFF_ID || ''

export function LiffProvider({ children }) {
  const [state, setState] = useState({
    ready: false,
    loggedIn: false,
    lineProfile: null,
    user: null,
    tenant_id: null,
    error: null,
  })

  useEffect(() => {
    const tenantId = new URLSearchParams(window.location.search).get('tenant_id')
      || localStorage.getItem('liff_tenant_id')
    if (tenantId) localStorage.setItem('liff_tenant_id', tenantId)

    const init = async () => {
      try {
        if (LIFF_ID) {
          await liff.init({ liffId: LIFF_ID })
        }

        // If already have a valid token, restore session
        const savedToken = localStorage.getItem('liff_token')
        const savedUser = localStorage.getItem('liff_user')
        if (savedToken && savedUser) {
          setState({
            ready: true,
            loggedIn: true,
            lineProfile: null,
            user: JSON.parse(savedUser),
            tenant_id: tenantId || localStorage.getItem('liff_tenant_id'),
            error: null,
          })
          return
        }

        // In real LIFF env: auto-login if already authenticated
        if (LIFF_ID && liff.isInClient() && liff.isLoggedIn()) {
          await loginWithLine(tenantId)
          return
        }

        setState(s => ({ ...s, ready: true, tenant_id: tenantId }))
      } catch (err) {
        setState(s => ({ ...s, ready: true, error: err.message }))
      }
    }

    init()
  }, [])

  const loginWithLine = async (tenantId) => {
    const tid = tenantId || localStorage.getItem('liff_tenant_id')
    if (!tid) {
      setState(s => ({ ...s, error: 'ไม่พบ tenant_id ในลิงก์' }))
      return
    }
    try {
      let lineUserId, name
      if (LIFF_ID && liff.isLoggedIn()) {
        const profile = await liff.getProfile()
        lineUserId = profile.userId
        name = profile.displayName
      } else {
        // Dev mode mock — stable ID across refreshes
        lineUserId = localStorage.getItem('dev_line_id') || (() => {
          const id = 'dev_' + Math.random().toString(36).slice(2, 10)
          localStorage.setItem('dev_line_id', id)
          return id
        })()
        name = 'Dev User'
      }

      const res = await client.post('/auth/line', { line_user_id: lineUserId, name, tenant_id: tid })
      const { token, user } = res.data
      localStorage.setItem('liff_token', token)
      localStorage.setItem('liff_user', JSON.stringify(user))

      setState({
        ready: true,
        loggedIn: true,
        lineProfile: { userId: lineUserId, displayName: name },
        user,
        tenant_id: tid,
        error: null,
      })
    } catch (err) {
      setState(s => ({ ...s, error: err.response?.data?.error || err.message }))
    }
  }

  const triggerLineLogin = () => {
    const tid = state.tenant_id || localStorage.getItem('liff_tenant_id')
    if (LIFF_ID && !liff.isLoggedIn()) {
      liff.login({ redirectUri: window.location.href })
    } else {
      loginWithLine(tid)
    }
  }

  const logout = () => {
    localStorage.removeItem('liff_token')
    localStorage.removeItem('liff_user')
    localStorage.removeItem('liff_branch')
    if (LIFF_ID && liff.isLoggedIn()) liff.logout()
    setState(s => ({ ...s, loggedIn: false, user: null, lineProfile: null }))
  }

  return (
    <LiffContext.Provider value={{ ...state, triggerLineLogin, logout }}>
      {children}
    </LiffContext.Provider>
  )
}

export const useLiff = () => useContext(LiffContext)
