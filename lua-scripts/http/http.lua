function handle(message)
    return "Hello Http Lua! Your message:"..message..", was received from "..getRemoteAddr().."!"
end