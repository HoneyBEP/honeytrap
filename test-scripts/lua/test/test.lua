--
-- Created by IntelliJ IDEA.
-- User: nordi
-- Date: 12-6-2018
-- Time: 14:21
-- To change this template use File | Settings | File Templates.
--
function canHandle(message)
    if (message == "pass") then
        return true
    elseif (message == "fail") then
        return false
    end

    return true
end
