$largePassword = "A" * 1100000

$largeBody = @{
    username = "admin"
    password = $largePassword
    role     = "admin"
} | ConvertTo-Json

try {
    Invoke-WebRequest `
        -Method Post `
        -Uri "http://localhost:8080/api/auth/login" `
        -ContentType "application/json" `
        -Body $largeBody
}
catch {
    [int]$_.Exception.Response.StatusCode
}