import './htmx.js'
import 'htmx.org/dist/ext/ws.js'
import {marked} from 'marked'
import katex from 'katex'

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

const renderer = new marked.Renderer()

function mathsExpression(expr) {
    if (expr.match(/\$\$[\s\S]*\$\$/)) {
        console.log("------ $$")
        expr = expr.substring(2, expr.length - 4)
        return katex.renderToString(expr, { displayMode: true })
    } else if (expr.match(/\$[\s\S]*\$/)) {
        console.log("------ $")
        expr = expr.substring(1, expr.length - 2)
        return katex.renderToString(expr, { displayMode: false })
    }
    console.log("------")
}

const rendererCode = renderer.code
renderer.code = function(code) {
    if (!code.lang) {
        const math = mathsExpression(code.text)
        if (math) {
            return math
        }
    }
    return rendererCode(code)
}

const rendererCodespan = renderer.codespan
renderer.codespan = function(codespan) {
    const math = mathsExpression(codespan.text)
    if (math) {
        return math
    }
    return rendererCodespan(codespan)
}

function parseRawMessage(message) {
    const raw = message.querySelector('.raw')
    const formatted = message.querySelector('.formatted')

    if (formatted.innerHTML === "") {
        const s = atob(raw.innerHTML)

        if (message.classList.contains('user')) {
            formatted.innerHTML = s
        } else {
            // Process think tags in assistant messages
            const thinkRegex = /<think>([\s\S]*?)<\/think>/g

            if (s.includes("<think>")) {
                let result = ""
                let lastIndex = 0
                let match

                // Reset the regex to start from the beginning
                thinkRegex.lastIndex = 0

                // Process each segment (alternating between regular and think content)
                while ((match = thinkRegex.exec(s)) !== null) {
                    // Add the regular content before this think block (parsed with markdown)
                    const regularContent = s.substring(lastIndex, match.index)
                    if (regularContent) {
                        result += marked.parse(regularContent, { renderer: renderer })
                    }

                    // Add the think content (parsed with markdown and wrapped in styled div)
                    const thinkContent = match[1];
                    result += `<div class="think-content">${marked.parse(thinkContent)}</div>`

                    // Update the last index to after this think block
                    lastIndex = match.index + match[0].length
                }

                // Add any remaining regular content after the last think block
                if (lastIndex < s.length) {
                    const remainingContent = s.substring(lastIndex);
                    result += marked(remainingContent, { renderer: renderer })
                }

                formatted.innerHTML = result
            } else {
                // No think tags, just parse the whole content with markdown
                formatted.innerHTML = marked.parse(s, { renderer: renderer })
            }
        }
    }
}
