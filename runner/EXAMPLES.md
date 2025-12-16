# Ejemplos de Uso del Runner

## Ejemplo 1: Verificar Versión de Terraform

### Request
```json
{
  "tool": "terraform",
  "work_dir": "/tmp",
  "command": "version",
  "args": [],
  "env_vars": {},
  "timeout": 5
}
```

### Comando
```bash
echo '{
  "tool": "terraform",
  "work_dir": "/tmp",
  "command": "version",
  "args": [],
  "env_vars": {},
  "timeout": 5
}' | ./iac-runner
```

### Response Esperado
```json
{
  "success": true,
  "output": "Terraform v1.14.2\non linux_amd64",
  "exit_code": 0,
  "error_msg": "",
  "started_at": "2025-12-16T10:00:00Z",
  "ended_at": "2025-12-16T10:00:01Z"
}
```

## Ejemplo 2: Terraform Init

### Request
```json
{
  "tool": "terraform",
  "work_dir": "/path/to/terraform/code",
  "command": "init",
  "args": [],
  "env_vars": {
    "TF_LOG": "INFO"
  },
  "timeout": 10
}
```

### Comando
```bash
echo '{
  "tool": "terraform",
  "work_dir": "/path/to/terraform/code",
  "command": "init",
  "args": [],
  "env_vars": {
    "TF_LOG": "INFO"
  },
  "timeout": 10
}' | ./iac-runner
```

## Ejemplo 3: Terraform Plan con Variables

### Request
```json
{
  "tool": "terraform",
  "work_dir": "/workspace/infrastructure",
  "command": "plan",
  "args": ["-out=tfplan", "-var-file=prod.tfvars"],
  "env_vars": {
    "TF_VAR_region": "us-east-1",
    "TF_VAR_environment": "production",
    "AWS_ACCESS_KEY_ID": "AKIAIOSFODNN7EXAMPLE",
    "AWS_SECRET_ACCESS_KEY": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
  },
  "timeout": 15
}
```

### Uso desde Backend (Go)
```go
request := RunnerRequest{
    Tool:    "terraform",
    WorkDir: "/workspace/infrastructure",
    Command: "plan",
    Args:    []string{"-out=tfplan", "-var-file=prod.tfvars"},
    EnvVars: map[string]string{
        "TF_VAR_region":      "us-east-1",
        "TF_VAR_environment": "production",
        "AWS_ACCESS_KEY_ID":     credentials.AccessKey,
        "AWS_SECRET_ACCESS_KEY": credentials.SecretKey,
    },
    Timeout: 15,
}

requestJSON, _ := json.Marshal(request)
cmd := exec.Command("/usr/local/bin/iac-runner")
cmd.Stdin = bytes.NewReader(requestJSON)

var stdout bytes.Buffer
cmd.Stdout = &stdout

cmd.Run()

var response RunnerResponse
json.Unmarshal(stdout.Bytes(), &response)

if response.Success {
    fmt.Println("Plan ejecutado exitosamente")
    fmt.Println(response.Output)
} else {
    fmt.Printf("Error: %s\n", response.ErrorMsg)
}
```

## Ejemplo 4: Terraform Apply

### Request
```json
{
  "tool": "terraform",
  "work_dir": "/workspace/infrastructure",
  "command": "apply",
  "args": ["tfplan"],
  "env_vars": {
    "AWS_ACCESS_KEY_ID": "AKIAIOSFODNN7EXAMPLE",
    "AWS_SECRET_ACCESS_KEY": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
  },
  "timeout": 30
}
```

## Ejemplo 5: OpenTofu Version

### Request
```json
{
  "tool": "tofu",
  "work_dir": "/tmp",
  "command": "version",
  "args": [],
  "env_vars": {},
  "timeout": 5
}
```

### Comando
```bash
echo '{
  "tool": "tofu",
  "work_dir": "/tmp",
  "command": "version",
  "args": [],
  "env_vars": {},
  "timeout": 5
}' | ./iac-runner
```

## Ejemplo 6: Terraform Destroy

### Request
```json
{
  "tool": "terraform",
  "work_dir": "/workspace/infrastructure",
  "command": "destroy",
  "args": ["-auto-approve"],
  "env_vars": {
    "TF_VAR_region": "us-east-1",
    "AWS_ACCESS_KEY_ID": "AKIAIOSFODNN7EXAMPLE",
    "AWS_SECRET_ACCESS_KEY": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
  },
  "timeout": 30
}
```

## Ejemplo 7: Manejo de Errores

### Request con Directorio Inválido
```json
{
  "tool": "terraform",
  "work_dir": "/non/existent/path",
  "command": "init",
  "args": [],
  "env_vars": {},
  "timeout": 5
}
```

### Response de Error
```json
{
  "success": false,
  "output": "",
  "exit_code": 1,
  "error_msg": "chdir /non/existent/path: no such file or directory",
  "started_at": "2025-12-16T10:00:00Z",
  "ended_at": "2025-12-16T10:00:00Z"
}
```

## Ejemplo 8: Timeout

### Request con Timeout Corto
```json
{
  "tool": "terraform",
  "work_dir": "/workspace/large-infrastructure",
  "command": "plan",
  "args": [],
  "env_vars": {},
  "timeout": 1
}
```

### Response
```json
{
  "success": false,
  "output": "Partial output before timeout...",
  "exit_code": 1,
  "error_msg": "signal: killed",
  "started_at": "2025-12-16T10:00:00Z",
  "ended_at": "2025-12-16T10:01:00Z"
}
```

## Ejemplo 9: Testing del Runner

### Script de Test Completo
```bash
#!/bin/bash

# Test 1: Terraform version
echo "Test 1: Terraform version"
echo '{"tool":"terraform","work_dir":"/tmp","command":"version","args":[],"env_vars":{},"timeout":5}' | ./iac-runner
echo ""

# Test 2: OpenTofu version
echo "Test 2: OpenTofu version"
echo '{"tool":"tofu","work_dir":"/tmp","command":"version","args":[],"env_vars":{},"timeout":5}' | ./iac-runner
echo ""

# Test 3: Invalid tool
echo "Test 3: Invalid tool (should fail)"
echo '{"tool":"invalid","work_dir":"/tmp","command":"version","args":[],"env_vars":{},"timeout":5}' | ./iac-runner
echo ""

# Test 4: Missing work_dir
echo "Test 4: Missing work_dir (should fail)"
echo '{"tool":"terraform","work_dir":"","command":"version","args":[],"env_vars":{},"timeout":5}' | ./iac-runner
```

## Ejemplo 10: Integración con Backend

### Función Helper en Go
```go
func runTerraformCommand(tool, workDir, command string, args []string, envVars map[string]string) (string, error) {
    request := RunnerRequest{
        Tool:    tool,
        WorkDir: workDir,
        Command: command,
        Args:    args,
        EnvVars: envVars,
        Timeout: 30,
    }

    requestJSON, err := json.Marshal(request)
    if err != nil {
        return "", fmt.Errorf("failed to marshal request: %w", err)
    }

    runnerPath := "/usr/local/bin/iac-runner"
    cmd := exec.Command(runnerPath)
    cmd.Stdin = bytes.NewReader(requestJSON)

    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    if err := cmd.Run(); err != nil {
        var response RunnerResponse
        if jsonErr := json.Unmarshal(stdout.Bytes(), &response); jsonErr == nil {
            return response.Output, fmt.Errorf("%s", response.ErrorMsg)
        }
        return stderr.String(), fmt.Errorf("runner execution failed: %w", err)
    }

    var response RunnerResponse
    if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
        return "", fmt.Errorf("failed to parse runner response: %w", err)
    }

    if !response.Success {
        return response.Output, fmt.Errorf("%s", response.ErrorMsg)
    }

    return response.Output, nil
}

// Uso
output, err := runTerraformCommand(
    "terraform",
    "/workspace/infra",
    "plan",
    []string{"-out=tfplan"},
    map[string]string{
        "TF_VAR_region": "us-east-1",
    },
)
```

## Tips y Mejores Prácticas

1. **Timeouts**: Ajusta el timeout según la complejidad del deployment
   - `init`: 5-10 minutos
   - `plan`: 10-20 minutos
   - `apply`: 20-60 minutos

2. **Variables de Entorno**: Usa `TF_VAR_*` para variables de Terraform
   ```json
   "env_vars": {
     "TF_VAR_instance_type": "t2.micro",
     "TF_VAR_ami_id": "ami-12345678"
   }
   ```

3. **Logging**: Habilita logs de Terraform para debugging
   ```json
   "env_vars": {
     "TF_LOG": "DEBUG",
     "TF_LOG_PATH": "/tmp/terraform.log"
   }
   ```

4. **Manejo de Credenciales**: Pasa credenciales como variables de entorno
   ```json
   "env_vars": {
     "AWS_ACCESS_KEY_ID": "...",
     "AWS_SECRET_ACCESS_KEY": "...",
     "ARM_CLIENT_ID": "...",
     "ARM_CLIENT_SECRET": "..."
   }
   ```

5. **Validación de Requests**: El runner valida automáticamente:
   - Tool debe ser "terraform" o "tofu"
   - WorkDir debe estar presente
   - Command debe estar presente
   - Timeout por defecto es 30 minutos
