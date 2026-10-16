param([Parameter(Mandatory=$true)][string]$ClientAddress)
$ErrorActionPreference='Stop'
New-Item -ItemType Directory -Force C:\ProgramData\Packer | Out-Null
Start-Transcript -Path C:\ProgramData\Packer\setup.log -Append | Out-Null
try {
  Copy-Item (Join-Path $PSScriptRoot 'verify-install.ps1') C:\ProgramData\Packer\verify-install.ps1 -Force
  Enable-PSRemoting -Force -SkipNetworkProfileCheck
  # Fresh Windows installations disable unencrypted WinRM by default. Verify
  # that invariant instead of rewriting it: on Win11 Public networks even a
  # redundant Set-Item can fail with the WinRM firewall-profile exception.
  if ([Convert]::ToBoolean((Get-Item WSMan:\localhost\Service\AllowUnencrypted).Value)) {
    throw 'Unencrypted WinRM must be disabled before using this HTTPS fixture'
  }
  Get-ChildItem WSMan:\localhost\Listener | Where-Object { $_.Keys -contains 'Transport=HTTPS' } | Remove-Item -Recurse -Force
  $cert=New-SelfSignedCertificate -DnsName $env:COMPUTERNAME -CertStoreLocation Cert:\LocalMachine\My
  New-Item -Path WSMan:\localhost\Listener -Transport HTTPS -Address * -CertificateThumbPrint $cert.Thumbprint -Force | Out-Null
  Set-Service WinRM -StartupType Automatic
  Get-NetFirewallRule -Name 'WINRM-HTTP-In-TCP*' -ErrorAction SilentlyContinue | Disable-NetFirewallRule
  New-NetFirewallRule -DisplayName 'Packer acceptance HTTPS WinRM' -Direction Inbound -Action Allow -Protocol TCP -LocalPort 5986 -RemoteAddress $ClientAddress -Profile Any | Out-Null
  Get-ChildItem WSMan:\localhost\Listener | Where-Object { $_.Keys -contains 'Transport=HTTP' } | Remove-Item -Recurse -Force
  New-Item -ItemType File -Force C:\ProgramData\Packer\installed | Out-Null
  & C:\ProgramData\Packer\verify-install.ps1
} finally {
  Stop-Transcript | Out-Null
}
