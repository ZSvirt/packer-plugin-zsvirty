#!/usr/bin/env python3
"""Render measured ISO acceptance stages; incomplete stages are never reported as passes."""
import argparse
from datetime import datetime
from html import escape
import json
import re
from pathlib import Path

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('reports', nargs='+', type=Path)
parser.add_argument('--output', type=Path, default=Path('.acceptance/iso-report.html'))
args = parser.parse_args()

def dt(value):
    value = re.sub(r'\.(\d+)', lambda m: '.' + (m.group(1) + '000000')[:6], value)
    return datetime.fromisoformat(value.replace('Z', '+00:00'))

def seconds(start, end):
    return f'{(dt(end) - dt(start)).total_seconds():.2f}'

def elapsed_with_minutes(start, end):
    elapsed = (dt(end) - dt(start)).total_seconds()
    minutes, remainder = divmod(round(elapsed), 60)
    return f'{elapsed:.2f}（{minutes} 分 {remainder:02d} 秒）'

def stages(report):
    steps = report['steps']
    def find(name):
        return next((s for s in steps if s['name'].endswith('.'+name)), None)
    def span(first, last):
        if first and last and first['completed'] and last['completed']:
            return seconds(first['started'], last['finished'])
        return '未完成'
    added, ready = find('StepAddImage'), find('StepWaitForImageReady')
    created, connected = find('StepCreateVMInstance'), find('StepConnect')
    image, exported = find('StepCreateImage'), find('StepExportImage')
    return [
        span(added, ready) if added else '复用已有 ISO',
        span(created, connected),
        seconds(connected['finished'], image['started']) if connected and connected['completed'] and image else '未完成',
        span(image, exported) if report.get('export_enabled') else '未启用导出',
        seconds(report['build_finished'], report['finished']) if report['passed'] else '未完成',
        seconds(report['started'], report['finished']) + ('' if report['passed'] else '（中断运行耗时）' if report.get('interrupted') else '（失败运行耗时）'),
        elapsed_with_minutes(report['started'], exported['finished']) if exported and exported['completed'] and report.get('export_enabled') else '未完成或未启用导出',
        ('人工干预诊断，不计自动验收通过' if report.get('manual_intervention') else '通过' if report['passed'] else '已中断，未完成验收' if report.get('interrupted') else '未通过'),
        report.get('image_uuid') or '无',
        report.get('notes') or '—',
    ]

rows = ['把 ISO 添加到镜像存储，等待 Ready', '创建云主机、自动安装系统，直到 SSH/WinRM 连接成功',
        '卸载安装盘、验证硬盘重启和文件传输，然后关机', '用根盘制作镜像，等待 Ready 并导出',
        '用新镜像创建云主机再次验证，并清理测试资源', '构建及镜像复验总时间', '开始构建到镜像导出成功', '验收结论', '保留镜像 UUID', '备注']
reports = [(p.stem.removesuffix('-timing'), json.loads(p.read_text())) for p in args.reports]
columns = [stages(r) for _, r in reports]
html = '<!doctype html><html lang="zh-CN"><meta charset="utf-8"><title>ISO 验收阶段耗时</title>'
html += '<style>body{font:15px system-ui;margin:32px;color:#202124}table{border-collapse:collapse;width:100%}th,td{border-bottom:1px solid #e5e7eb;padding:14px;text-align:left;vertical-align:top}th:first-child,td:first-child{min-width:260px}th{background:#f8fafc}tbody tr:nth-child(6),tbody tr:nth-child(7){font-weight:700}p{color:#555}td{overflow-wrap:anywhere}</style>'
html += '<h1>ISO 验收阶段耗时</h1><p>单位：秒。仅使用本次测试实测数据；“未完成”不计为通过。总时间包含前置校验、应答光盘制作、登录及清理开销，因此可能大于阶段耗时之和。复用已有 ISO 不计导入耗时。</p><table><thead><tr><th>阶段</th>'
html += ''.join('<th>'+escape(r.get('label') or name)+'</th>' for name, r in reports)+'</tr></thead><tbody>'
for i, name in enumerate(rows):
    html += '<tr><th>'+escape(name)+'</th>'+''.join('<td>'+escape(c[i])+'</td>' for c in columns)+'</tr>'
html += '</tbody></table><p>计时从临时磁盘规格创建后开始，到镜像复验和 VM 清理后结束；不含 Go 编译和临时磁盘规格创建、删除。ARM64 需 ARM64 宿主机；截图中的示例数值未作为本环境测试结果。</p></html>'
args.output.parent.mkdir(parents=True, exist_ok=True)
args.output.write_text(html)
print(args.output.resolve())
