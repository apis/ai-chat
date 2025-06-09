import './htmx.js'
import 'htmx.org/dist/ext/ws.js'
import {marked} from 'marked'

import "@fontsource/noto-sans/100.css"
import "@fontsource/noto-sans/200.css"
import "@fontsource/noto-sans/300.css"
import "@fontsource/noto-sans/400.css"
import "@fontsource/noto-sans/500.css"
import "@fontsource/noto-sans/600.css"
import "@fontsource/noto-sans/700.css"
import "@fontsource/noto-sans/800.css"
import "@fontsource/noto-sans/900.css"

import "@fontsource/noto-sans-mono/100.css"
import "@fontsource/noto-sans-mono/200.css"
import "@fontsource/noto-sans-mono/300.css"
import "@fontsource/noto-sans-mono/400.css"
import "@fontsource/noto-sans-mono/500.css"
import "@fontsource/noto-sans-mono/600.css"
import "@fontsource/noto-sans-mono/700.css"
import "@fontsource/noto-sans-mono/800.css"
import "@fontsource/noto-sans-mono/900.css"

import './style.css'

window.chatWrapperScrollAutomatically = false

window.mainContentScrollToBottom = function(){
    if (window.chatWrapperScrollAutomatically === true) {
        const chatWrapper = document.querySelector(".main-content")
        chatWrapper.scrollTop = chatWrapper.scrollHeight
        window.chatWrapperScrollAutomatically = false
    }
}

document.body.addEventListener("htmx:oobBeforeSwap", function (evt) {
    const chatMessages = document.querySelector('#main .chat-messages')
    const lastAssistantChatMessage = document.querySelector('#main .chat-message.assistant:last-child')

    if (evt.detail.target === chatMessages || evt.detail.target === lastAssistantChatMessage) {
        const chatWrapper = document.querySelector(".main-content")

        if (Math.abs(chatWrapper.scrollHeight - chatWrapper.clientHeight - chatWrapper.scrollTop) <= 1) {
            window.chatWrapperScrollAutomatically = true
        }
    }
})

document.body.addEventListener("htmx:oobAfterSwap", function (evt) {
    const chatMessages = document.querySelector('#main .chat-messages')
    const lastAssistantChatMessage = document.querySelector('#main .chat-message.assistant:last-child')

    if (evt.detail.target === chatMessages) {
        const messages = chatMessages.querySelectorAll('.chat-message')

        for (const message of messages) {
            parseRawMessage(message)
        }
        window.mainContentScrollToBottom()

    } else if (evt.detail.target === lastAssistantChatMessage) {
        parseRawMessage(lastAssistantChatMessage)
        window.mainContentScrollToBottom()
    }
})

document.body.addEventListener("parseAllRawMessages", function(evt){
    const messages = document.querySelectorAll('#main .chat-message')

    for (const message of messages) {
        parseRawMessage(message)
    }

    window.chatWrapperScrollAutomatically = true
    window.mainContentScrollToBottom()
})

document.body.addEventListener("clearUserInput", function(evt){
    const userInput = document.querySelector('.main-footer .user-input')
    userInput.value = ''
})

function parseRawMessage(message) {
    const raw = message.querySelector('.raw')
    const formatted = message.querySelector('.formatted')

    if (formatted.innerHTML === "") {
        const s = atob(raw.innerHTML)

        if (message.classList.contains('user')) {
            formatted.innerHTML = s
        } else {
            formatted.innerHTML = marked.parse(s)
        }
    }
}
