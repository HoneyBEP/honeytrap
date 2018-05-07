----------------------------------------
-- canHandle function returns the approval or disapproval to handle the connection. This is based on the information
-- of the current connection and new request. The service is already determined by the Honeytrap implementation in Go.
----------------------------------------
function canHandle()

end

----------------------------------------
-- preHandle function is called by the desired service and pre handles the request by the connection. This function can
-- be used to alter, log or visualize the incoming request in the Honeytrap. From the Honeytrap implementation in Go
-- there are protocol or service specific functions declared. Take a look in the Honeytrap documentation to check the
-- availability for standard services. When implementing your own service you can freely implement the preHandle
-- function in your service.
----------------------------------------
function preHandle()

end

----------------------------------------
-- handle function is called by the desired service and handles the request by the connection. From the Honeytrap
-- implementation in Go there are protocol or service specific functions declared. Take a look in the Honeytrap
-- documentation to check the availability for standard services. When implementing your own service you can freely
-- implement the handle function in your service.
----------------------------------------
function handle(message)
    return "Hello Http Lua! Your message:"..message..", was received from ".. getRemoteAddr() .." on ".. getDatetime() .."!"
end

----------------------------------------
-- afterHandle function is called by the desired service and after handles the request by the connection. This function
-- can be used to alter, log or visualize the outgoing response in the Honeytrap. From the Honeytrap implementation in
-- Go there are protocol or service specific functions declared. Take a look in the Honeytrap documentation to check the
-- availability for standard services. When implementing your own service you can freely implement the afterHandle
-- function in your service.
----------------------------------------
function afterHandle()

end