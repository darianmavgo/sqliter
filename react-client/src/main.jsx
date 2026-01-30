import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.jsx'
import { initLogger } from './logger.js'

console.log("Main.jsx executing - attempting to verify logging connection");
initLogger();
console.log("Logger initialized from Main.jsx");

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
