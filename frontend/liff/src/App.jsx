import React from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { LiffProvider, useLiff } from './contexts/LiffContext.jsx'
import Login from './pages/Login.jsx'
import BranchSelect from './pages/BranchSelect.jsx'
import Upload from './pages/Upload.jsx'
import Status from './pages/Status.jsx'
import Conversation from './pages/Conversation.jsx'
import BottomNav from './components/BottomNav.jsx'

function ProtectedRoutes() {
  const { ready, loggedIn } = useLiff()
  if (!ready) return <Spinner />
  if (!loggedIn) return <Navigate to="/login" replace />
  return (
    <div className="flex flex-col min-h-screen">
      <div className="flex-1 pb-16">
        <Routes>
          <Route path="/branch" element={<BranchSelect />} />
          <Route path="/upload" element={<Upload />} />
          <Route path="/status" element={<Status />} />
          <Route path="/conversation" element={<Conversation />} />
          <Route path="*" element={
            localStorage.getItem('liff_branch')
              ? <Navigate to="/upload" replace />
              : <Navigate to="/branch" replace />
          } />
        </Routes>
      </div>
      <BottomNav />
    </div>
  )
}

function Spinner() {
  return (
    <div className="flex items-center justify-center min-h-screen bg-white">
      <div className="w-10 h-10 border-4 border-line border-t-transparent rounded-full animate-spin" />
    </div>
  )
}

export default function App() {
  return (
    <LiffProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/*" element={<ProtectedRoutes />} />
        </Routes>
      </BrowserRouter>
    </LiffProvider>
  )
}
