#!/usr/bin/env python3
"""jinja2templ: translate Basecoat's Jinja macros into templ components.

    jinja2templ.py <jinja-dir> <out-dir>              translate every *.html.jinja
    jinja2templ.py --testdata <jinja-dir> <out-dir>   render testdata/cases.json with jinja2

Each macro becomes a templ component in package `jinja` (one .templ per
.jinja). The translation is mechanical: Jinja's AST (jinja2's own parser)
is walked with a small HTML lexer that carries its state across template
data, so a `{% if %}` inside a start tag becomes a templ conditional
attribute, an `{% if %}` inside a quoted value becomes a jif() expression,
and `{% for k, v in d.items() %}{{ k }}="{{ v }}"{% endfor %}` becomes a
spread. Values with no static type go through a small runtime (rt.go,
written from RUNTIME below) that keeps Jinja's semantics: truthiness,
str(), `default`, `tojson`, `in`, nil-safe attribute access.

What the translator can't do it writes as the identifier TODO_jinja2templ
with a comment, which stops the package compiling until a person fixes it,
and counts it (stats.json in the out dir).
"""
import json
import os
import re
import sys

from jinja2 import Environment, nodes

ENV = Environment(autoescape=True)
GO_KEYWORDS = {"break", "case", "chan", "const", "continue", "default", "defer", "else",
               "fallthrough", "for", "func", "go", "goto", "if", "import", "interface",
               "map", "package", "range", "return", "select", "struct", "switch", "type",
               "var", "templ", "len", "string", "any", "ctx", "children"}

GO_TYPE = {"string": "string", "bool": "bool", "int": "int", "any": "any",
           "attrs": "templ.Attributes", "list": "[]any", "dict": "map[string]any",
           "component": "templ.Component"}


def camel(name, export=False):
    parts = [p for p in name.split("_") if p]
    s = parts[0] + "".join(p[:1].upper() + p[1:] for p in parts[1:])
    if export:
        return s[:1].upper() + s[1:]
    if s in GO_KEYWORDS:
        s += "_"
    return s


def attr_context(name):
    """templ's attribute contexts (internal/htmlattr.Classify), for its sanitiser."""
    n = name.lower()
    if n.startswith(("on", "hx-on", "data-hx-on")):
        return "js"
    if n == "style":
        return "css"
    n = n.removeprefix("data-") if n.startswith("data-") else n.split(":")[-1]
    if n in ("action", "archive", "background", "cite", "classid", "codebase", "data", "formaction",
             "href", "icon", "longdesc", "manifest", "poster", "profile", "src", "usemap", "xmlns") \
            or "src" in n or "uri" in n or "url" in n:
        return "url"
    return ""


def golit(s):
    return json.dumps(s, ensure_ascii=False)


class Todo(Exception):
    def __init__(self, kind, detail=""):
        super().__init__(kind)
        self.kind, self.detail = kind, detail


# --- signatures --------------------------------------------------------

class Sig:
    def __init__(self, macro, file):
        self.macro, self.file = macro, file
        self.name = camel(macro.name, export=True)
        self.params = []  # (jinja name, go name, type, default node or None)
        self.caller = "none"  # none | children | param


def walk(node):
    yield node
    for child in node.iter_child_nodes():
        yield from walk(child)


def default_type(d):
    if d is None:
        return None
    if isinstance(d, nodes.Const):
        v = d.value
        if isinstance(v, bool):
            return "bool"
        if isinstance(v, int):
            return "int"
        if isinstance(v, str):
            return "string"
        return None
    if isinstance(d, nodes.Dict):
        return "attrs"
    if isinstance(d, nodes.List):
        return "list"
    return None


def usage_type(macro, pname, sigs):
    """The type a None-defaulted (or undefaulted) parameter is used as."""
    t, lst = "string", False
    for n in walk(macro):
        if isinstance(n, nodes.For) and isinstance(n.iter, nodes.Name) and n.iter.name == pname:
            lst = True
        if isinstance(n, nodes.Call) and isinstance(n.node, nodes.Name) and n.node.name in sigs:
            callee = sigs[n.node.name]
            for i, a in enumerate(n.args):
                if isinstance(a, nodes.Name) and a.name == pname and i < len(callee.params) \
                        and callee.params[i][2] == "list":
                    lst = True
        if isinstance(n, (nodes.Getattr, nodes.Getitem)) and isinstance(n.node, nodes.Name) \
                and n.node.name == pname:
            t = "any"
        if isinstance(n, nodes.Test) and isinstance(n.node, nodes.Name) and n.node.name == pname \
                and n.name != "defined":
            t = "any"
        if isinstance(n, nodes.Compare) and any(o.op in ("in", "notin") and isinstance(o.expr, nodes.Name)
                                                and o.expr.name == pname for o in n.ops):
            t = "any"
        if isinstance(n, nodes.Filter) and n.name in ("tojson",) and isinstance(n.node, nodes.Name) \
                and n.node.name == pname:
            t = "any"
    # dynamic use (attributes, tests) wins over iteration: jlist() iterates an any
    return "any" if t == "any" else ("list" if lst else t)


def caller_use(macro):
    """children: caller() only, or `caller() if caller is defined`; param: tested on its own."""
    calls = tests = guarded = 0
    for n in walk(macro):
        if isinstance(n, nodes.Call) and isinstance(n.node, nodes.Name) and n.node.name == "caller":
            calls += 1
        if isinstance(n, nodes.Test) and n.name == "defined" and isinstance(n.node, nodes.Name) \
                and n.node.name == "caller":
            tests += 1
        if isinstance(n, nodes.CondExpr) and is_call(n.expr1, "caller") and is_caller_test(n.test):
            guarded += 1
    if not calls and not tests:
        return "none"
    if tests == guarded:
        return "children"
    return "param"


def is_call(n, name):
    return isinstance(n, nodes.Call) and isinstance(n.node, nodes.Name) and n.node.name == name


def is_caller_test(n):
    return isinstance(n, nodes.Test) and n.name == "defined" and isinstance(n.node, nodes.Name) \
        and n.node.name == "caller"


def signatures(macros):
    sigs = {}
    for m, f in macros:
        s = Sig(m, f)
        defaults = [None] * (len(m.args) - len(m.defaults)) + list(m.defaults)
        for a, d in zip(m.args, defaults):
            s.params.append([a.name, camel(a.name), default_type(d), d])
        s.caller = caller_use(m)
        sigs[m.name] = s
    for _ in range(2):  # twice, so a list passed on to a callee's list param is seen
        for s in sigs.values():
            for p in s.params:
                dt = default_type(p[3])
                if dt is None:
                    p[2] = usage_type(s.macro, p[0], sigs)
    return sigs


# --- expressions -------------------------------------------------------

def unify(a, b):
    if a == b:
        return a
    if {a, b} == {"attrs", "dict"}:
        return "attrs"
    return "any"


def coerce(code, typ, want):
    if typ == want:
        return code
    if want == "any":
        if typ in ("dict", "any"):
            return code
        return f"any({code})"
    if want == "string":
        return f"jstr({code})"
    if want == "bool":
        if typ == "string":
            return f"({code} != \"\")"
        return f"jtrue({code})"
    if want == "int":
        return f"jint({code})"
    if want == "attrs":
        if typ == "dict":
            return f"templ.Attributes({code})"
        return f"jattrsOf({code})"
    if want == "list":
        return f"jlist({code})"
    if want == "dict":
        return f"jdict({code})"
    raise Todo("coercion", f"{typ}->{want}")


class Scope:
    def __init__(self, parent=None, loop=None):
        self.parent, self.vars, self.loop = parent, {}, loop

    def find(self, name):
        s = self
        while s:
            if name in s.vars:
                return s.vars[name]
            s = s.parent
        return None

    def find_loop(self):
        s = self
        while s:
            if s.loop:
                return s.loop
            s = s.parent
        return None


class Macro:
    """Translates one macro's body."""

    def __init__(self, sig, sigs, stats):
        self.sig, self.sigs, self.stats = sig, sigs, stats
        self.buf = []
        self.loops = 0
        self.st = "text"  # text | tag | aname | preval | aval
        self.aname = ""
        self.quote = ""
        self.parts = []
        self.todos = []
        self.mech = 0

    # bookkeeping
    def todo(self, kind, detail, lineno):
        self.todos.append((kind, detail, lineno))
        return f"TODO_jinja2templ /* {kind}: {detail} (line {lineno}) */"

    def w(self, s):
        self.buf.append(s)

    # expressions: return (go code, type)
    def x(self, n, sc):
        m = getattr(self, "x_" + type(n).__name__, None)
        if m is None:
            raise Todo("expression", type(n).__name__)
        return m(n, sc)

    def xs(self, n, sc, want):
        code, typ = self.x(n, sc)
        return coerce(code, typ, want)

    def x_Const(self, n, sc):
        v = n.value
        if v is None:
            return "nil", "any"
        if isinstance(v, bool):
            return ("true" if v else "false"), "bool"
        if isinstance(v, int):
            return str(v), "int"
        if isinstance(v, str):
            return golit(v), "string"
        raise Todo("constant", repr(v))

    def x_Name(self, n, sc):
        v = sc.find(n.name)
        if v is None:
            if n.name in ("true", "false"):
                return n.name, "bool"
            raise Todo("unknown name", n.name)
        return v

    def x_Getattr(self, n, sc):
        if isinstance(n.node, nodes.Name) and n.node.name == "loop" and sc.find("loop") is None:
            lp = sc.find_loop()
            if lp is None:
                raise Todo("loop outside for", n.attr)
            if n.attr == "index":
                return f"({lp}+1)", "int"
            if n.attr == "index0":
                return lp, "int"
            if n.attr == "first":
                return f"({lp} == 0)", "bool"
            raise Todo("loop attribute", n.attr)
        return f"jget({self.xs(n.node, sc, 'any')}, {golit(n.attr)})", "any"

    def x_Getitem(self, n, sc):
        base = self.xs(n.node, sc, "any")
        if isinstance(n.arg, nodes.Const) and isinstance(n.arg.value, str):
            return f"jget({base}, {golit(n.arg.value)})", "any"
        return f"jindex({base}, {self.xs(n.arg, sc, 'any')})", "any"

    def x_Filter(self, n, sc):
        name = n.name
        if n.kwargs or n.dyn_args or n.dyn_kwargs:
            raise Todo("filter arguments", name)
        if name == "safe":
            return self.x(n.node, sc)
        if name == "default":
            if len(n.args) > 1:
                raise Todo("default(boolean=)", "")
            code, typ = self.x(n.node, sc)
            d, dt = self.x(n.args[0], sc) if n.args else ('""', "string")
            if typ != "any":  # a typed value is never undefined
                return code, typ
            return f"jdefault({code}, {coerce(d, dt, 'any')})", "any"
        if name == "string":
            return self.xs(n.node, sc, "string"), "string"
        if name == "random":
            if is_call(n.node, "range") and len(n.node.args) == 2:
                a = self.xs(n.node.args[0], sc, "int")
                b = self.xs(n.node.args[1], sc, "int")
                return f"jrandrange({a}, {b})", "int"
            return f"jrandom({self.xs(n.node, sc, 'list')})", "any"
        if name == "tojson":
            return f"jtojson({self.xs(n.node, sc, 'any')})", "string"
        if name == "length":
            return f"jlen({self.xs(n.node, sc, 'any')})", "int"
        if name == "join":
            sep = self.xs(n.args[0], sc, "string") if n.args else '""'
            return f"jjoin({self.xs(n.node, sc, 'any')}, {sep})", "string"
        raise Todo("filter", name)

    def x_Test(self, n, sc):
        if n.name == "defined":
            if isinstance(n.node, nodes.Name):
                if n.node.name == "caller":
                    if self.sig.caller == "param":
                        return "(caller != nil)", "bool"
                    raise Todo("caller test", "")
                return "true", "bool"
            return f"({self.xs(n.node, sc, 'any')} != nil)", "bool"
        code, typ = self.x(n.node, sc)
        if n.name == "none":
            if typ != "any":
                return "false", "bool"
            return f"({code} == nil)", "bool"
        if n.name == "string":
            return f"jisstring({coerce(code, typ, 'any')})", "bool"
        if n.name == "iterable":
            return f"jiterable({coerce(code, typ, 'any')})", "bool"
        raise Todo("test", n.name)

    def x_Not(self, n, sc):
        return f"!{self.xs(n.node, sc, 'bool')}", "bool"

    def _andor(self, n, sc, op, fn):
        a, at = self.x(n.left, sc)
        b, bt = self.x(n.right, sc)
        if self.want_bool:
            return f"({coerce(a, at, 'bool')} {op} {coerce(b, bt, 'bool')})", "bool"
        t = unify(at, bt)
        if t == "bool":
            return f"({a} {op} {b})", "bool"
        return f"{fn}[{GO_TYPE[t]}]({coerce(a, at, t)}, {coerce(b, bt, t)})", t

    want_bool = False

    def xb(self, n, sc):
        """An expression in a boolean context."""
        old, self.want_bool = self.want_bool, True
        try:
            if isinstance(n, (nodes.And, nodes.Or)):
                code, typ = self.x(n, sc)
            else:
                self.want_bool = False
                code, typ = self.x(n, sc)
            return coerce(code, typ, "bool")
        finally:
            self.want_bool = old

    def x_And(self, n, sc):
        if self.want_bool:
            return f"({self.xb(n.left, sc)} && {self.xb(n.right, sc)})", "bool"
        return self._andor(n, sc, "&&", "jand")

    def x_Or(self, n, sc):
        if self.want_bool:
            return f"({self.xb(n.left, sc)} || {self.xb(n.right, sc)})", "bool"
        return self._andor(n, sc, "||", "jor")

    def x_Compare(self, n, sc):
        if len(n.ops) != 1:
            raise Todo("chained comparison", "")
        old, self.want_bool = self.want_bool, False
        try:
            a, at = self.x(n.expr, sc)
            op = n.ops[0]
            b, bt = self.x(op.expr, sc)
        finally:
            self.want_bool = old
        if op.op in ("in", "notin"):
            r = f"jin({coerce(a, at, 'any')}, {coerce(b, bt, 'any')})"
            return (r if op.op == "in" else "!" + r), "bool"
        goop = {"eq": "==", "ne": "!=", "gt": ">", "gteq": ">=", "lt": "<", "lteq": "<="}[op.op]
        if at == bt and at in ("string", "int", "bool") or {at, bt} == {"int"}:
            return f"({a} {goop} {b})", "bool"
        if op.op in ("eq", "ne"):
            r = f"jeq({coerce(a, at, 'any')}, {coerce(b, bt, 'any')})"
            return (r if op.op == "eq" else "!" + r), "bool"
        return f"(jcmp({coerce(a, at, 'any')}, {coerce(b, bt, 'any')}) {goop} 0)", "bool"

    def x_Concat(self, n, sc):
        return "(" + " + ".join(self.xs(p, sc, "string") for p in n.nodes) + ")", "string"

    def x_Add(self, n, sc):
        a, at = self.x(n.left, sc)
        b, bt = self.x(n.right, sc)
        if at == bt and at in ("string", "int"):
            return f"({a} + {b})", at
        return f"jadd({coerce(a, at, 'any')}, {coerce(b, bt, 'any')})", "any"

    def x_CondExpr(self, n, sc):
        c = self.xb(n.test, sc)
        a, at = self.x(n.expr1, sc)
        if n.expr2 is None:  # undefined: renders as ""
            b, bt = ('""', "string") if at == "string" else ("nil", "any")
        else:
            b, bt = self.x(n.expr2, sc)
        t = unify(at, bt)
        if at == "any" and b == "nil":
            t = "any"
        return f"jif[{GO_TYPE[t]}]({c}, {coerce(a, at, t)}, {coerce(b, bt, t)})", t

    def x_List(self, n, sc):
        return "[]any{" + ", ".join(self.xs(i, sc, "any") for i in n.items) + "}", "list"

    def x_Tuple(self, n, sc):
        return self.x_List(n, sc)

    def x_Dict(self, n, sc):
        items = []
        for p in n.items:
            items.append(f"{self.xs(p.key, sc, 'string')}: {self.xs(p.value, sc, 'any')}")
        return "map[string]any{" + ", ".join(items) + "}", "dict"

    def x_Call(self, n, sc):
        if n.dyn_args or n.dyn_kwargs:
            raise Todo("*args/**kwargs call", "")
        f = n.node
        if isinstance(f, nodes.Name):
            if f.name == "namespace":
                items = [f"{golit(k.key)}: {self.xs(k.value, sc, 'any')}" for k in n.kwargs]
                return "map[string]any{" + ", ".join(items) + "}", "dict"
            if f.name == "caller":
                if n.args or n.kwargs:
                    raise Todo("caller with args", "")
                if self.sig.caller == "param":
                    return "@caller", "component"
                return "{ children... }", "component"
            if f.name in self.sigs:
                return self.macro_call(self.sigs[f.name], n, sc), "component"
            raise Todo("call", f.name)
        if isinstance(f, nodes.Getattr):
            if f.attr == "get" and len(n.args) in (1, 2) and not n.kwargs:
                base = self.xs(f.node, sc, "any")
                k = self.xs(n.args[0], sc, "string")
                d = self.xs(n.args[1], sc, "any") if len(n.args) == 2 else "nil"
                return f"jgetd({base}, {k}, {d})", "any"
            if f.attr == "join" and isinstance(f.node, nodes.Const) and len(n.args) == 1:
                return f"jjoin({self.xs(n.args[0], sc, 'any')}, {golit(f.node.value)})", "string"
            raise Todo("method call", f.attr)
        raise Todo("call", type(f).__name__)

    def macro_call(self, callee, n, sc):
        args = {}
        for i, a in enumerate(n.args):
            args[callee.params[i][0]] = a
        for k in n.kwargs:
            if k.key not in [p[0] for p in callee.params]:
                raise Todo("kwargs", k.key)
            args[k.key] = k.value
        out = []
        for jname, _, typ, d in callee.params:
            v = args.get(jname, d)
            if v is None:
                out.append({"string": '""', "bool": "false", "int": "0"}.get(typ, "nil"))
            else:
                out.append(self.xs(v, sc, typ))
        if callee.caller == "param":
            out.append("nil")
        return f"@{callee.name}({', '.join(out)})"

    # --- statements -----------------------------------------------------

    def body(self, ns, sc):
        for n in ns:
            self.stmt(n, sc)

    def stmt(self, n, sc):
        try:
            m = getattr(self, "s_" + type(n).__name__, None)
            if m is None:
                raise Todo("statement", type(n).__name__)
            m(n, sc)
        except Todo as t:
            self.w(" " + self.todo(t.kind, t.detail, n.lineno) + " ")

    def count(self):
        self.mech += 1

    def s_Output(self, n, sc):
        for p in n.nodes:
            if isinstance(p, nodes.TemplateData):
                self.text(p.data, p.lineno)
            else:
                try:
                    self.output(p, sc)
                    self.count()
                except Todo as t:
                    self.boundary()
                    if self.st == "aval":
                        self.parts.append(("e", self.todo(t.kind, t.detail, p.lineno)))
                    elif self.st == "text":
                        self.w("{ " + self.todo(t.kind, t.detail, p.lineno) + " }")
                    else:
                        self.w("\n{ " + self.todo(t.kind, t.detail, p.lineno) + "... }\n")

    def output(self, e, sc):
        if self.st == "aval":
            self.parts.append(("e", self.xs(e, sc, "string")))
            return
        self.boundary()
        if self.st == "tag":
            # {{ "inert" if cond }}: a conditional bare attribute
            if isinstance(e, nodes.CondExpr) and isinstance(e.expr1, nodes.Const) and e.expr2 is None \
                    and isinstance(e.expr1.value, str) and re.fullmatch(r"[\w-]+", e.expr1.value):
                self.w(f"\nif {self.xb(e.test, sc)} {{\n{e.expr1.value}\n}}\n")
                return
            raise Todo("expression in a start tag", type(e).__name__)
        if self.st != "text":
            raise Todo("expression in attribute name", "")
        safe = isinstance(e, nodes.Filter) and e.name == "safe"
        if isinstance(e, nodes.CondExpr) and e.expr2 is None:
            if is_call(e.expr1, "caller") and is_caller_test(e.test) and self.sig.caller == "children":
                self.w("\n{ children... }\n")
                return
            code, typ = self.x(e.expr1, sc)
            if typ == "component":
                self.w(f"\nif {self.xb(e.test, sc)} {{\n{code}\n}}\n")
                return
        if isinstance(e, nodes.CondExpr) and is_call(e.expr1, "caller") and is_caller_test(e.test) \
                and isinstance(e.expr2, nodes.Const) and e.expr2.value == "" and self.sig.caller == "children":
            self.w("\n{ children... }\n")
            return
        code, typ = self.x(e, sc)
        if typ == "component":
            self.w(f"\n{code}\n")
        elif safe:
            self.w(f"\n@templ.Raw({coerce(code, typ, 'string')})\n")
        else:
            self.w("{ " + coerce(code, typ, "string") + " }")

    def value_expr(self, ns, sc):
        """The Go string expression for a run of nodes inside a quoted attribute value."""
        parts = []
        for n in ns:
            if isinstance(n, nodes.Output):
                for p in n.nodes:
                    if isinstance(p, nodes.TemplateData):
                        if self.quote in p.data:
                            raise Todo("quote inside a conditional value", "")
                        parts.append(golit(p.data))
                    else:
                        parts.append(self.xs(p, sc, "string"))
            elif isinstance(n, nodes.If):
                parts.append(self.if_value(n, sc))
            else:
                raise Todo("statement inside an attribute value", type(n).__name__)
        return " + ".join(parts) if parts else '""'

    def if_value(self, n, sc):
        then = self.value_expr(n.body, sc)
        if n.elif_:
            rest = nodes.If(n.elif_[0].test, n.elif_[0].body, n.elif_[1:], n.else_, lineno=n.lineno)
            other = self.if_value(rest, sc)
        else:
            other = self.value_expr(n.else_, sc)
        return f"jif[string]({self.xb(n.test, sc)}, {then}, {other})"

    def s_If(self, n, sc):
        self.boundary()
        if self.st == "aval":
            self.parts.append(("e", self.if_value(n, sc)))
            self.count()
            return
        if self.st not in ("text", "tag"):
            raise Todo("if in attribute name", "")
        st = self.st
        self.w(f"\nif {self.xb(n.test, sc)} {{\n")
        self.body(n.body, sc)
        self.branch_end(st, n)
        for e in n.elif_:
            self.w(f"\n}} else if {self.xb(e.test, sc)} {{\n")
            self.body(e.body, sc)
            self.branch_end(st, e)
        if n.else_:
            self.w("\n} else {\n")
            self.body(n.else_, sc)
            self.branch_end(st, n)
        self.w("\n}\n")
        self.count()

    def branch_end(self, st, n):
        self.boundary()
        if self.st != st:
            self.w(" " + self.todo("if branch leaves the HTML state changed", f"{st}->{self.st}", n.lineno))
            self.st = st

    def attr_loop(self, n, sc):
        """{% for k, v in d.items() %}[{% if k != 'x' and ... %}]{{ k }}="{{ v }}"[{% endif %}]{% endfor %}"""
        t = n.target
        if not (isinstance(t, nodes.Tuple) and len(t.items) == 2 and isinstance(n.iter, nodes.Call)
                and isinstance(n.iter.node, nodes.Getattr) and n.iter.node.attr == "items"):
            return None
        k, v = t.items[0].name, t.items[1].name
        body = [b for b in n.body if not (isinstance(b, nodes.Output) and all(
            isinstance(p, nodes.TemplateData) and not p.data.strip() for p in b.nodes))]
        excl = []
        if len(body) == 1 and isinstance(body[0], nodes.If) and not body[0].elif_ and not body[0].else_:
            excl = self.exclusions(body[0].test, k)
            if excl is None:
                return None
            body = body[0].body
        if len(body) != 1 or not isinstance(body[0], nodes.Output):
            return None
        ps = [p for p in body[0].nodes if not (isinstance(p, nodes.TemplateData) and not p.data.strip())]
        if not (len(ps) == 3 and isinstance(ps[0], nodes.Name) and ps[0].name == k
                and isinstance(ps[1], nodes.TemplateData) and ps[1].data.strip() == '="'
                and isinstance(ps[2], nodes.Name) and ps[2].name == v):
            # {{ k }}="{{ v }}" lexes as Name, Data('="'), Name, Data('"')
            ps = [p for p in body[0].nodes]
            ps = [p for p in ps if not (isinstance(p, nodes.TemplateData) and not p.data.strip())]
            if not (len(ps) == 4 and isinstance(ps[0], nodes.Name) and ps[0].name == k
                    and ps[1].data.strip() == '="' and isinstance(ps[2], nodes.Name) and ps[2].name == v
                    and ps[3].data.strip() == '"'):
                return None
        d = self.xs(n.iter.node.node, sc, "any")
        return f"jattrs({', '.join([d] + [golit(e) for e in excl])})"

    def exclusions(self, test, k):
        if isinstance(test, nodes.And):
            a, b = self.exclusions(test.left, k), self.exclusions(test.right, k)
            return None if a is None or b is None else a + b
        if isinstance(test, nodes.Compare) and len(test.ops) == 1 and test.ops[0].op == "ne" \
                and isinstance(test.expr, nodes.Name) and test.expr.name == k \
                and isinstance(test.ops[0].expr, nodes.Const):
            return [test.ops[0].expr.value]
        return None

    def s_For(self, n, sc):
        self.boundary()
        if n.else_ or n.test or n.recursive:
            raise Todo("for with else/if/recursive", "")
        if self.st == "tag":
            spread = self.attr_loop(n, sc)
            if spread is None:
                raise Todo("for in a start tag", "")
            self.w(f"\n{{ {spread}... }}\n")
            self.count()
            return
        if self.st != "text":
            raise Todo("for inside an attribute", "")
        if not isinstance(n.target, nodes.Name):
            raise Todo("for with a tuple target", "")
        self.loops += 1
        used = {x.name for x in walk(nodes.Output(n.body)) if isinstance(x, nodes.Name)}
        idx = f"loop{self.loops}" if "loop" in used else "_"
        var = camel(n.target.name) if n.target.name in used else "_"
        inner = Scope(sc, loop=idx if idx != "_" else None)
        inner.vars[n.target.name] = (var, "any")
        self.w(f"\nfor {idx}, {var} := range {self.xs(n.iter, sc, 'list')} {{\n")
        self.hoist(n.body, inner)
        self.body(n.body, inner)
        self.boundary()
        self.w("\n}\n")
        self.count()

    def hoist(self, body, sc):
        """Jinja's {% if %} doesn't open a scope and Go's does: declare first, as any."""
        seen = set()

        def scan(ns, nested):
            for n in ns:
                if isinstance(n, nodes.Assign) and isinstance(n.target, nodes.Name):
                    name = n.target.name
                    if nested and name not in seen and sc.find(name) is None:
                        sc.vars[name] = (camel(name), "any")
                        self.w(f"\n{{{{ var {camel(name)} any }}}}\n{{{{ _ = {camel(name)} }}}}\n")
                    seen.add(name)
                elif isinstance(n, nodes.If):
                    scan(n.body, True)
                    for e in n.elif_:
                        scan(e.body, True)
                    scan(n.else_, True)
        scan(body, False)

    def s_Assign(self, n, sc):
        self.boundary()
        if self.st != "text":
            raise Todo("set inside a tag", "")
        t = n.target
        if isinstance(t, nodes.NSRef):
            ns = sc.find(t.name)
            if ns is None:
                raise Todo("unknown namespace", t.name)
            self.w(f"\n{{{{ {ns[0]}[{golit(t.attr)}] = {self.xs(n.node, sc, 'any')} }}}}\n")
            self.count()
            return
        if not isinstance(t, nodes.Name):
            raise Todo("set with a tuple target", "")
        v = sc.vars.get(t.name)
        if v is not None:  # in this scope (a param, a hoisted var, an earlier set): assign
            self.w(f"\n{{{{ {v[0]} = {self.xs(n.node, sc, v[1])} }}}}\n")
        else:  # new here (Jinja's for body is a scope, so this may shadow)
            code, typ = self.x(n.node, sc)
            if typ == "int":
                typ = "any"
            if typ == "component":
                raise Todo("set to a macro call", "")
            name = camel(t.name)
            self.w(f"\n{{{{ {name} := {coerce(code, typ, typ)} }}}}\n{{{{ _ = {name} }}}}\n")
            sc.vars[t.name] = (name, typ)
        self.count()

    # --- the HTML lexer -------------------------------------------------

    def attr_start(self):
        """A quoted attribute value begins (a hook for templapi.py)."""

    def boundary(self):
        if self.st == "aname":
            self.emit_attr(self.aname, None)
            self.st = "tag"

    def emit_attr(self, name, parts):
        if parts is None:
            self.w(f"\n{name}")
            return
        if all(k == "s" for k, _ in parts):
            val = "".join(v for _, v in parts)
            q = '"' if '"' not in val else "'"
            self.w(f"\n{name}={q}{val}{q}")
            return
        code, lit = [], ""
        for k, v in parts:
            if k == "s":
                lit += v
            else:
                if lit:
                    code.append(golit(lit))
                    lit = ""
                code.append(v)
        if lit:
            code.append(golit(lit))
        expr = " + ".join(code)
        # Jinja trusts what the caller passes; templ sanitises by context, so
        # say the value is trusted, as the macro does (NOTICE.md).
        ctx = attr_context(name)
        if ctx == "js":
            expr = f"templ.JSUnsafeFuncCall({expr})"
        elif ctx == "url":
            expr = f"templ.SafeURL({expr})"
        elif ctx == "css":
            expr = f"templ.SafeCSS({expr})"
        self.w(f"\n{name}={{ {expr} }}")

    def text(self, s, lineno):
        i = 0
        while i < len(s):
            c = s[i]
            if self.st == "text":
                if c == "<" and i + 1 < len(s) and (s[i + 1].isalpha()):
                    m = re.match(r"<([a-zA-Z][\w-]*)", s[i:])
                    self.w("<" + m.group(1))
                    i += len(m.group(0))
                    self.st = "tag"
                    continue
                if c == "<" and s.startswith("</", i):
                    j = s.index(">", i)
                    self.w(s[i:j + 1])
                    i = j + 1
                    continue
                if c in "{}":
                    self.w('{ "' + c + '" }')
                else:
                    self.w(c)
                i += 1
            elif self.st == "tag":
                if c.isspace():
                    i += 1
                elif c == ">":
                    self.w("\n>")
                    self.st = "text"
                    i += 1
                elif s.startswith("/>", i):
                    self.w("\n/>")
                    self.st = "text"
                    i += 2
                else:
                    self.st, self.aname = "aname", ""
            elif self.st == "aname":
                if c.isspace() or c in ">/":
                    self.boundary()
                elif c == "=":
                    self.st = "preval"
                    i += 1
                else:
                    self.aname += c
                    i += 1
            elif self.st == "preval":
                if c in "\"'":
                    self.st, self.quote, self.parts = "aval", c, []
                    self.attr_start()
                    i += 1
                else:
                    self.w(" " + self.todo("unquoted attribute value", self.aname, lineno))
                    self.st = "tag"
            elif self.st == "aval":
                if c == self.quote:
                    self.emit_attr(self.aname, self.parts)
                    self.st = "tag"
                else:
                    if self.parts and self.parts[-1][0] == "s":
                        self.parts[-1] = ("s", self.parts[-1][1] + c)
                    else:
                        self.parts.append(("s", c))
                i += 1

    # --- the component ----------------------------------------------------

    def translate(self):
        s = self.sig
        sc = Scope()
        params = []
        for jname, gname, typ, _ in s.params:
            sc.vars[jname] = (gname, typ)
            params.append(f"{gname} {GO_TYPE[typ]}")
        if s.caller == "param":
            params.append("caller templ.Component")
        self.w(f"// {s.name} is Basecoat's {s.macro.name} macro ({s.file}:{s.macro.lineno}).\n")
        self.w(f"templ {s.name}({', '.join(params)}) {{\n")
        self.hoist(s.macro.body, sc)
        self.body(s.macro.body, sc)
        self.boundary()
        if self.st != "text":
            self.w(" " + self.todo("macro ends inside a tag", self.st, s.macro.lineno))
        self.w("\n}\n")
        # tidy: drop runs of blank lines
        out = "".join(self.buf)
        out = re.sub(r"[ \t]+\n", "\n", out)
        out = re.sub(r"\n\s*\n+", "\n", out)
        return out


# --- runtime ---------------------------------------------------------------

RUNTIME = r'''// Code generated by internal/jinja/gen/jinja2templ.py; DO NOT EDIT.

// The Jinja semantics the
// translated macros (*.templ here) need, on Go values (string, bool, int,
// float64, nil, []any, map[string]any, templ.Attributes).
package jinja

import (
	"fmt"
	"math/rand/v2"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/a-h/templ"
)

// jtrue is Python's truthiness.
func jtrue(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	case int:
		return x != 0
	case float64:
		return x != 0
	case []any:
		return len(x) > 0
	case map[string]any:
		return len(x) > 0
	case templ.Attributes:
		return len(x) > 0
	case templ.Component:
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Map:
		return rv.Len() > 0
	}
	return true
}

// jstr is how {{ v }} prints v (nil is Undefined: "").
func jstr(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case bool:
		if x {
			return "True"
		}
		return "False"
	case int:
		return strconv.Itoa(x)
	case float64:
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'g', -1, 64)
	}
	return fmt.Sprint(v)
}

func jint(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case float64:
		return int(x)
	case bool:
		if x {
			return 1
		}
	case string:
		n, _ := strconv.Atoi(x)
		return n
	}
	return 0
}

func jmap(v any) map[string]any {
	switch x := v.(type) {
	case map[string]any:
		return x
	case templ.Attributes:
		return x
	}
	return nil
}

// jget is d.k / d['k']: nil when d isn't a dict or has no k.
func jget(d any, k string) any {
	if m := jmap(d); m != nil {
		return m[k]
	}
	return nil
}

// jgetd is d.get(k, def).
func jgetd(d any, k string, def any) any {
	if m := jmap(d); m != nil {
		if v, ok := m[k]; ok {
			return v
		}
	}
	return def
}

func jindex(l any, i any) any {
	if s, ok := l.([]any); ok {
		n := jint(i)
		if n < 0 {
			n += len(s)
		}
		if n >= 0 && n < len(s) {
			return s[n]
		}
		return nil
	}
	if s, ok := i.(string); ok {
		return jget(l, s)
	}
	return nil
}

// jdefault is the default filter: def when v is undefined.
func jdefault(v any, def any) any {
	if v == nil {
		return def
	}
	return v
}

func jlist(v any) []any {
	switch x := v.(type) {
	case nil:
		return nil
	case []any:
		return x
	case string:
		out := make([]any, 0, len(x))
		for _, r := range x {
			out = append(out, string(r))
		}
		return out
	}
	if m := jmap(v); m != nil {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make([]any, len(keys))
		for i, k := range keys {
			out[i] = k
		}
		return out
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Slice {
		out := make([]any, rv.Len())
		for i := range out {
			out[i] = rv.Index(i).Interface()
		}
		return out
	}
	return nil
}

func jdict(v any) map[string]any { return jmap(v) }

func jattrsOf(v any) templ.Attributes { return templ.Attributes(jmap(v)) }

// jattrs is {% for k, v in d.items() %}{% if k != x... %}{{ k }}="{{ v }}"{% endif %}{% endfor %}:
// every value printed as Jinja prints it (True, not a bare attribute).
func jattrs(d any, exclude ...string) templ.Attributes {
	m := jmap(d)
	out := make(templ.Attributes, len(m))
outer:
	for k, v := range m {
		for _, x := range exclude {
			if k == x {
				continue outer
			}
		}
		s := "None"
		if v != nil {
			s = jstr(v)
		}
		// Jinja doesn't sanitise what the caller passes; templ does, by
		// context, unless the value says it is trusted.
		switch lk := strings.ToLower(k); {
		case strings.HasPrefix(lk, "on"), strings.HasPrefix(lk, "hx-on"), strings.HasPrefix(lk, "data-hx-on"):
			out[k] = templ.JSUnsafeFuncCall(s)
		case lk == "style":
			out[k] = templ.SafeCSS(s)
		default:
			out[k] = templ.SafeURL(s)
		}
	}
	return out
}

func jlen(v any) int {
	switch x := v.(type) {
	case nil:
		return 0
	case string:
		return len([]rune(x))
	}
	if m := jmap(v); m != nil {
		return len(m)
	}
	return len(jlist(v))
}

func jjoin(v any, sep string) string {
	var parts []string
	for _, x := range jlist(v) {
		parts = append(parts, jstr(x))
	}
	return strings.Join(parts, sep)
}

func jrandrange(lo, hi int) int { return lo + rand.IntN(hi-lo) }

func jrandom(l []any) any {
	if len(l) == 0 {
		return nil
	}
	return l[rand.IntN(len(l))]
}

func jnum(v any) (float64, bool) {
	switch x := v.(type) {
	case int:
		return float64(x), true
	case float64:
		return x, true
	case bool:
		if x {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

// jeq is Python's ==.
func jeq(a, b any) bool {
	if x, ok := jnum(a); ok {
		if y, ok := jnum(b); ok {
			return x == y
		}
		return false
	}
	if sa, ok := a.(string); ok {
		sb, ok := b.(string)
		return ok && sa == sb
	}
	return reflect.DeepEqual(a, b)
}

func jcmp(a, b any) int {
	if x, ok := jnum(a); ok {
		if y, ok := jnum(b); ok {
			switch {
			case x < y:
				return -1
			case x > y:
				return 1
			}
			return 0
		}
	}
	return strings.Compare(jstr(a), jstr(b))
}

// jin is Python's in.
func jin(x, c any) bool {
	switch y := c.(type) {
	case nil:
		return false
	case string:
		s, ok := x.(string)
		return ok && strings.Contains(y, s)
	}
	if m := jmap(c); m != nil {
		s, ok := x.(string)
		if !ok {
			return false
		}
		_, ok = m[s]
		return ok
	}
	for _, e := range jlist(c) {
		if jeq(x, e) {
			return true
		}
	}
	return false
}

// jadd is Python's + on values with no static type.
func jadd(a, b any) any {
	if x, ok := a.(string); ok {
		return x + jstr(b)
	}
	if x, ok := jnum(a); ok {
		y, _ := jnum(b)
		if _, fa := a.(float64); !fa {
			if _, fb := b.(float64); !fb {
				return int(x + y)
			}
		}
		return x + y
	}
	l := append([]any{}, jlist(a)...)
	return append(l, jlist(b)...)
}

// jor and jand are Python's or and and: they return an operand.
func jor[T any](a, b T) T {
	if jtrue(any(a)) {
		return a
	}
	return b
}

func jand[T any](a, b T) T {
	if !jtrue(any(a)) {
		return a
	}
	return b
}

// jif is a if c else b (both sides evaluated: every j* accessor is nil-safe).
func jif[T any](c bool, a, b T) T {
	if c {
		return a
	}
	return b
}

func jiterable(v any) bool {
	switch v.(type) {
	case string, []any, map[string]any, templ.Attributes:
		return true
	}
	return reflect.ValueOf(v).Kind() == reflect.Slice
}

func jisstring(v any) bool {
	_, ok := v.(string)
	return ok
}

// jtojson is the tojson filter: json.dumps(sort_keys=True) with <>&' escaped.
func jtojson(v any) string {
	var b strings.Builder
	jsonw(&b, v)
	r := strings.NewReplacer("<", `<`, ">", `>`, "&", `&`, "'", `'`)
	return r.Replace(b.String())
}

func jsonw(b *strings.Builder, v any) {
	switch x := v.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		if x {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case int, float64:
		b.WriteString(jstr(x))
	case string:
		b.WriteByte('"')
		for _, r := range x {
			switch {
			case r == '"':
				b.WriteString(`\"`)
			case r == '\\':
				b.WriteString(`\\`)
			case r == '\n':
				b.WriteString(`\n`)
			case r == '\r':
				b.WriteString(`\r`)
			case r == '\t':
				b.WriteString(`\t`)
			case r < 0x20 || r > 0x7e:
				if r > 0xffff {
					r -= 0x10000
					fmt.Fprintf(b, `\u%04x\u%04x`, 0xd800+(r>>10), 0xdc00+(r&0x3ff))
				} else {
					fmt.Fprintf(b, `\u%04x`, r)
				}
			default:
				b.WriteRune(r)
			}
		}
		b.WriteByte('"')
	default:
		if m := jmap(v); m != nil {
			keys := make([]string, 0, len(m))
			for k := range m {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			b.WriteByte('{')
			for i, k := range keys {
				if i > 0 {
					b.WriteString(", ")
				}
				jsonw(b, k)
				b.WriteString(": ")
				jsonw(b, m[k])
			}
			b.WriteByte('}')
			return
		}
		b.WriteByte('[')
		for i, e := range jlist(v) {
			if i > 0 {
				b.WriteString(", ")
			}
			jsonw(b, e)
		}
		b.WriteByte(']')
	}
}
'''


# --- the dispatcher the test calls (generated from the signatures) --------

def dispatcher(sigs):
    out = ["// Code generated by internal/jinja/gen/jinja2templ.py; DO NOT EDIT.",
           "",
           "// Call calls a component by its macro name with Jinja-style keyword",
           "// arguments (missing ones take the macro's default), as testdata/cases.json",
           "// and ui's render tests give them.",
           "package jinja", "", 'import "github.com/a-h/templ"', "",
           "func Call(macro string, args map[string]any, caller templ.Component) templ.Component {",
           "\tswitch macro {"]
    for s in sigs.values():
        m = Macro(s, sigs, None)
        sc = Scope()
        conv = []
        for jname, gname, typ, d in s.params:
            if d is None:
                dflt = {"string": '""', "bool": "false", "int": "0"}.get(typ, "nil")
            else:
                dflt = m.xs(d, sc, typ)
            get = coerce(f"jgetd(args, {golit(jname)}, any({dflt}))", "any", typ)
            conv.append(get)
        if s.caller == "param":
            conv.append("caller")
        out.append(f"\tcase {golit(s.macro.name)}:")
        out.append(f"\t\treturn {s.name}({', '.join(conv)})")
    out += ["\t}", "\treturn nil", "}", ""]
    return "\n".join(out)


def load(jdir):
    files = sorted(f for f in os.listdir(jdir) if f.endswith(".html.jinja"))
    macros = []
    for f in files:
        src = open(os.path.join(jdir, f)).read()
        tree = ENV.parse(src)
        for m in tree.find_all(nodes.Macro):
            macros.append((m, f))
    return files, macros


def translate(jdir, odir):
    files, macros = load(jdir)
    sigs = signatures(macros)
    os.makedirs(odir, exist_ok=True)
    stats = {}
    for f in files:
        src = open(os.path.join(jdir, f)).read()
        base = f.removesuffix(".html.jinja").replace("-", "_")
        out = [f"// Code generated by internal/jinja/gen/jinja2templ.py from Basecoat 1.1.0's\n"
               f"// templates/jinja/{f} (MIT; NOTICE.md). DO NOT EDIT.\n",
               "package jinja\n"]
        fstat = {"source_lines": src.count("\n") + 1, "mechanical": 0, "todos": [], "macros": []}
        for m, mf in macros:
            if mf != f:
                continue
            tr = Macro(sigs[m.name], sigs, stats)
            out.append("\n" + tr.translate())
            fstat["mechanical"] += tr.mech
            fstat["todos"] += [{"kind": k, "detail": d, "line": ln, "macro": m.name} for k, d, ln in tr.todos]
            fstat["macros"].append({"macro": m.name, "component": sigs[m.name].name,
                                    "caller": sigs[m.name].caller,
                                    "params": [[p[1], GO_TYPE[p[2]]] for p in sigs[m.name].params]})
        text = "".join(out)
        open(os.path.join(odir, base + ".templ"), "w").write(text)
        fstat["templ_lines"] = text.count("\n")
        stats[f] = fstat
    open(os.path.join(odir, "rt.go"), "w").write(RUNTIME)
    open(os.path.join(odir, "call_gen.go"), "w").write(dispatcher(sigs))
    json.dump(stats, open(os.path.join(odir, "stats.json"), "w"), indent=1)
    for f, s in stats.items():
        print(f"{f:28} {s['source_lines']:4} lines  {s['mechanical']:3} constructs  {len(s['todos'])} TODO")


# --- testdata: the expected HTML, rendered by jinja2 -------------------------

def testdata(jdir, odir):
    import random
    random.seed(1)  # the macros' random ids, the same on every run
    cases = json.load(open(os.path.join(odir, "testdata", "cases.json")))
    for c in cases:
        src = open(os.path.join(jdir, c["file"])).read()
        mod = ENV.from_string(src).module
        fn = getattr(mod, c["macro"])
        if "caller" in c:
            html = ENV.from_string(
                "{% call m(**args) %}" + c["caller"] + "{% endcall %}").render(m=fn, args=c["args"])
        else:
            html = str(fn(**c["args"]))
        open(os.path.join(odir, "testdata", c["name"] + ".html"), "w").write(html)
        print("wrote", c["name"])


if __name__ == "__main__":
    if sys.argv[1] == "--testdata":
        testdata(sys.argv[2], sys.argv[3])
    else:
        translate(sys.argv[1], sys.argv[2])
