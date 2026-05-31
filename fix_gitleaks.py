with open('.gitleaks.toml', 'r') as f:
    content = f.read()

# Prepend [extend]\nuseDefault = true\n\n
content = '[extend]\nuseDefault = true\n\n' + content

with open('.gitleaks.toml', 'w') as out:
    out.write(content)
