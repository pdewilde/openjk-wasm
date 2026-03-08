import re

with open('codemp/ui/ui_shared.c', 'r') as f:
    content = f.read()

# Find all global functions
funcs = re.findall(r'^[a-zA-Z0-9_* ]+\s+([a-zA-Z0-9_]+)\s*\(', content, re.MULTILINE)
# Find all global variables
vars = re.findall(r'^[a-zA-Z0-9_* ]+\s+([a-zA-Z0-9_]+)\s*(?:\[.*?\])?\s*(?:=.*?)?;', content, re.MULTILINE)

symbols = set(funcs + vars)

# Filter out static and local stuff
filtered_symbols = []
for sym in symbols:
    if sym in ['if', 'while', 'for', 'switch', 'return', 'sizeof', 'int', 'void', 'char', 'float', 'qboolean', 'struct', 'with', 'WITHOUT', 'is']:
        continue
    if 'static' in sym:
        continue
    filtered_symbols.append(sym)

filtered_symbols.sort()

with open('codemp/ui/ui_rename.h', 'a') as f:
    f.write('\n// Auto-generated from ui_shared.c\n')
    for sym in filtered_symbols:
        f.write(f'#ifndef {sym}\n#define {sym} ui_{sym}\n#endif\n')

with open('codemp/cgame/cg_rename.h', 'a') as f:
    f.write('\n// Auto-generated from ui_shared.c\n')
    for sym in filtered_symbols:
        f.write(f'#ifndef {sym}\n#define {sym} cg_{sym}\n#endif\n')

print("Done!")
