import React, { useEffect, useRef, useState } from 'react'
import { useLiff } from '../contexts/LiffContext.jsx'
import client from '../api/client.js'

function MessageBubble({ msg }) {
  const isCustomer = msg.sender_type === 'customer'
  const time = new Date(msg.created_at).toLocaleTimeString('th-TH', { hour: '2-digit', minute: '2-digit' })
  return (
    <div className={`flex ${isCustomer ? 'justify-end' : 'justify-start'} mb-3`}>
      {!isCustomer && (
        <div className="w-8 h-8 rounded-full bg-line flex items-center justify-center text-white text-xs font-bold mr-2 flex-shrink-0 self-end">
          ทีม
        </div>
      )}
      <div className={`max-w-[75%]`}>
        <div className={`px-3.5 py-2.5 rounded-2xl text-sm ${
          isCustomer
            ? 'bg-line text-white rounded-br-sm'
            : 'bg-white border border-gray-200 text-gray-800 rounded-bl-sm'
        }`}>
          {msg.content}
        </div>
        <p className={`text-[10px] text-gray-400 mt-0.5 ${isCustomer ? 'text-right' : 'text-left'}`}>
          {time}
        </p>
      </div>
    </div>
  )
}

function ConversationThread({ conv, onBack }) {
  const [messages, setMessages] = useState([])
  const [text, setText] = useState('')
  const [sending, setSending] = useState(false)
  const bottomRef = useRef(null)

  useEffect(() => {
    client.get(`/conversations/${conv.id}/messages`)
      .then(res => setMessages(res.data.data || []))
  }, [conv.id])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const send = async () => {
    if (!text.trim() || sending) return
    setSending(true)
    try {
      const res = await client.post(`/conversations/${conv.id}/messages`, {
        message_type: 'text',
        content: text.trim(),
        sender_type: 'customer',
      })
      setMessages(prev => [...prev, res.data.data])
      setText('')
    } finally {
      setSending(false)
    }
  }

  const handleKey = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      send()
    }
  }

  return (
    <div className="flex flex-col h-screen">
      {/* Header */}
      <div className="flex items-center gap-3 px-4 py-3 bg-white border-b border-gray-200 safe-top">
        <button onClick={onBack} className="text-line">
          <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M15 19l-7-7 7-7" />
          </svg>
        </button>
        <div>
          <p className="font-semibold text-gray-800 text-sm">สนทนากับทีมบัญชี</p>
          <p className="text-xs text-gray-400">
            {conv.status === 'open' ? '🟢 เปิดอยู่' : '⚫ ปิดแล้ว'}
          </p>
        </div>
      </div>

      {/* Messages */}
      <div className="flex-1 overflow-y-auto px-4 py-4 bg-gray-50">
        {messages.length === 0 ? (
          <div className="text-center text-gray-400 text-sm mt-8">ยังไม่มีข้อความ</div>
        ) : (
          messages.map(m => <MessageBubble key={m.id} msg={m} />)
        )}
        <div ref={bottomRef} />
      </div>

      {/* Input */}
      {conv.status === 'open' && (
        <div className="flex items-end gap-2 px-4 py-3 bg-white border-t border-gray-200 safe-bottom">
          <textarea
            value={text}
            onChange={e => setText(e.target.value)}
            onKeyDown={handleKey}
            placeholder="พิมพ์ข้อความ..."
            rows={1}
            className="flex-1 resize-none border border-gray-300 rounded-xl px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-line max-h-32"
          />
          <button
            onClick={send}
            disabled={!text.trim() || sending}
            className="w-10 h-10 bg-line rounded-full flex items-center justify-center flex-shrink-0 disabled:opacity-40"
          >
            <svg className="w-5 h-5 text-white" fill="currentColor" viewBox="0 0 24 24">
              <path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z"/>
            </svg>
          </button>
        </div>
      )}
    </div>
  )
}

export default function Conversation() {
  const { user } = useLiff()
  const [conversations, setConversations] = useState([])
  const [loading, setLoading] = useState(true)
  const [active, setActive] = useState(null)

  useEffect(() => {
    client.get(`/conversations?tenant_id=${user?.tenant_id || ''}`)
      .then(res => setConversations(res.data.data || []))
      .finally(() => setLoading(false))
  }, [])

  if (active) {
    return <ConversationThread conv={active} onBack={() => setActive(null)} />
  }

  return (
    <div className="px-4 py-6 max-w-md mx-auto">
      <h1 className="text-xl font-bold text-gray-800 mb-5">ประวัติสนทนา</h1>

      {loading ? (
        <div className="flex justify-center py-16">
          <div className="w-8 h-8 border-4 border-line border-t-transparent rounded-full animate-spin" />
        </div>
      ) : conversations.length === 0 ? (
        <div className="text-center py-16">
          <div className="text-5xl mb-3">💬</div>
          <p className="text-gray-500 text-sm">ยังไม่มีประวัติการสนทนา</p>
          <p className="text-gray-400 text-xs mt-1">ทีมบัญชีจะติดต่อกลับผ่านช่องนี้</p>
        </div>
      ) : (
        <div className="space-y-3">
          {conversations.map(conv => {
            const date = new Date(conv.updated_at).toLocaleDateString('th-TH', {
              day: '2-digit', month: 'short',
            })
            return (
              <button
                key={conv.id}
                onClick={() => setActive(conv)}
                className="w-full text-left bg-white border border-gray-200 rounded-xl p-4 flex items-center justify-between"
              >
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-full bg-line flex items-center justify-center text-white text-sm font-bold">
                    ทีม
                  </div>
                  <div>
                    <p className="text-sm font-semibold text-gray-800">ทีมบัญชี</p>
                    <p className="text-xs text-gray-400">
                      {conv.status === 'open' ? '🟢 เปิดอยู่' : '⚫ ปิดแล้ว'} · {date}
                    </p>
                  </div>
                </div>
                <svg className="w-5 h-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                </svg>
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}
