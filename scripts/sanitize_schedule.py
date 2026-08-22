#!/usr/bin/env python3
"""
Sanitize a Microsoft Project CSV export by removing all identifying information
while preserving structural data needed for DCMA metric calculations.
"""

import csv
import sys
import re
from pathlib import Path

def sanitize_name(name, task_id):
    """Replace task name with generic anonymous name."""
    # Summary tasks
    if "Summaries & Milestones" in name:
        return "Project Summary"
    if "Project Management" in name:
        return "PM Phase"
    if "Main Milestones" in name:
        return "Key Milestones"
    if "Quality Gates" in name:
        return "Quality Checkpoints"
    if "Demarcatio/Customer Interfaces" in name or "Interfaces" in name:
        return "Interface Tasks"
    if "Payment Milestones" in name or "Invoicing" in name:
        return "Financial Phase"
    if "Installation Sub-Contractor" in name or "Sub-Contractor" in name:
        return "Contractor Phase"
    if "Permitting" in name:
        return "Permit Phase"
    if "Project Kick Off" in name or "Project Planning" in name:
        return "Planning Phase"
    if "Commodity Indexation" in name or "Hyper-Inflation" in name:
        return "Cost Adjustment Phase"
    if "Analytics Platform" in name:
        return "Analytics Phase"
    if "Engineering" in name:
        return "Engineering Phase"
    if "EHS" in name or "Safety" in name:
        return "Safety Phase"
    if "Mechanical Engineering" in name:
        return "Mechanical Phase"
    if "Controls Engineering" in name:
        return "Controls Phase"
    if "Software Engineering" in name:
        return "Software Phase"
    if "Procurement" in name:
        return "Procurement Phase"
    if "Manufacturing" in name or "Mfg" in name:
        return "Manufacturing Phase"
    if "Shipping" in name:
        return "Shipping Phase"
    if "Install" in name or "Installation" in name:
        return "Installation Phase"
    if "Commissioning" in name:
        return "Commissioning Phase"
    if "Testing" in name:
        return "Testing Phase"
    if "Acceptance" in name:
        return "Acceptance Phase"
    
    # Milestone tasks
    if "Milestone" in name or "Awarded" in name or "Complete" in name or "Approval" in name:
        return f"Milestone {task_id}"
    
    # Default: generic task name
    return f"Task {task_id}"

def sanitize_discipline(discipline):
    """Replace discipline with generic code."""
    if not discipline or discipline.strip() == "":
        return ""
    
    # Map to generic discipline codes
    discipline_lower = discipline.lower()
    if "mechanical" in discipline_lower:
        return "01 - Mechanical"
    if "controls" in discipline_lower or "electrical" in discipline_lower:
        return "02 - Controls"
    if "software" in discipline_lower:
        return "03 - Software"
    if "customer" in discipline_lower:
        return "04 - Client"
    if "project management" in discipline_lower or discipline_lower == "00":
        return "00 - Management"
    if "system engineer" in discipline_lower:
        return "05 - Systems"
    if "purchasing" in discipline_lower or "procurement" in discipline_lower:
        return "06 - Procurement"
    if "acceptance" in discipline_lower:
        return "07 - Acceptance"
    if "permitting" in discipline_lower:
        return "08 - Permitting"
    
    return "09 - General"

def sanitize_segment_nbr(segment_nbr):
    """Replace segment number with generic code."""
    if not segment_nbr or segment_nbr.strip() == "" or segment_nbr == "0":
        return ""
    
    # Map to generic segment codes
    try:
        num = int(segment_nbr)
        if num > 0:
            return f"SEG-{num % 100:03d}"
    except ValueError:
        pass
    return segment_nbr

def sanitize_segment_name(name):
    """Replace segment name with generic name."""
    if not name or name.strip() == "":
        return ""
    
    name_lower = name.lower()
    if "milestone" in name_lower:
        return "Milestone Group"
    if "quality gate" in name_lower:
        return "Quality Group"
    if "interface" in name_lower:
        return "Interface Group"
    if "invoicing" in name_lower:
        return "Financial Group"
    if "sub-contractor" in name_lower:
        return "Contractor Group"
    if "permitting" in name_lower:
        return "Permit Group"
    if "planning" in name_lower:
        return "Planning Group"
    if "mechanical" in name_lower:
        return "Mechanical Group"
    if "controls" in name_lower:
        return "Controls Group"
    if "procurement" in name_lower:
        return "Procurement Group"
    if "manufacturing" in name_lower:
        return "Manufacturing Group"
    if "shipping" in name_lower:
        return "Shipping Group"
    if "install" in name_lower:
        return "Installation Group"
    if "commissioning" in name_lower:
        return "Commissioning Group"
    if "testing" in name_lower:
        return "Testing Group"
    if "acceptance" in name_lower:
        return "Acceptance Group"
    
    return "General Group"

def sanitize_row(row, header_map):
    """Sanitize a single row of data."""
    sanitized = row.copy()
    
    # Get task ID for naming
    task_id = row[header_map.get('task_id', 0)] if 'task_id' in header_map else "0"
    
    # Sanitize name
    if 'name' in header_map:
        idx = header_map['name']
        if idx < len(sanitized):
            sanitized[idx] = sanitize_name(sanitized[idx], task_id)
    
    # Sanitize discipline
    if 'discipline' in header_map:
        idx = header_map['discipline']
        if idx < len(sanitized):
            sanitized[idx] = sanitize_discipline(sanitized[idx])
    
    # Sanitize mechanical segment number
    if 'mechanical_segment_nbr' in header_map:
        idx = header_map['mechanical_segment_nbr']
        if idx < len(sanitized):
            sanitized[idx] = sanitize_segment_nbr(sanitized[idx])
    
    # Sanitize control segment number
    if 'control_segment_nbr' in header_map:
        idx = header_map['control_segment_nbr']
        if idx < len(sanitized):
            sanitized[idx] = sanitize_segment_nbr(sanitized[idx])
    
    # Sanitize segment name
    if 'segment_name' in header_map:
        idx = header_map['segment_name']
        if idx < len(sanitized):
            sanitized[idx] = sanitize_segment_name(sanitized[idx])
    
    return sanitized

def main():
    if len(sys.argv) != 3:
        print(f"Usage: {sys.argv[0]} <input_csv> <output_csv>")
        sys.exit(1)
    
    input_path = Path(sys.argv[1])
    output_path = Path(sys.argv[2])
    
    if not input_path.exists():
        print(f"Error: Input file '{input_path}' not found")
        sys.exit(1)
    
    # Read input CSV
    with open(input_path, 'r', encoding='utf-8') as f:
        reader = csv.reader(f)
        rows = list(reader)
    
    if len(rows) < 2:
        print("Error: Input file has no data rows")
        sys.exit(1)
    
    # Parse header
    header = rows[0]
    header_map = {}
    for i, col in enumerate(header):
        # Normalize column name
        normalized = col.lower().replace(' ', '_').replace('-', '_')
        header_map[normalized] = i
        
        # Also map common variations
        if normalized == 'id':
            header_map['task_id'] = i
        if 'name' in normalized and 'segment' not in normalized:
            header_map['name'] = i
        if 'discipline' in normalized:
            header_map['discipline'] = i
        if 'mechanical' in normalized and 'segment' in normalized:
            header_map['mechanical_segment_nbr'] = i
        if 'control' in normalized and 'segment' in normalized:
            header_map['control_segment_nbr'] = i
        if 'segment' in normalized and 'name' in normalized:
            header_map['segment_name'] = i
        if 'segment' in normalized and 'type' in normalized:
            header_map['segment_type'] = i
    
    print(f"Found {len(header_map)} mappable columns")
    print(f"Total rows: {len(rows)} (1 header + {len(rows)-1} data rows)")
    
    # Sanitize data rows
    sanitized_rows = [header]  # Keep header as-is
    for i, row in enumerate(rows[1:], start=2):
        try:
            sanitized = sanitize_row(row, header_map)
            sanitized_rows.append(sanitized)
        except Exception as e:
            print(f"Warning: Error sanitizing row {i}: {e}")
            sanitized_rows.append(row)  # Keep original on error
    
    # Write output CSV
    with open(output_path, 'w', encoding='utf-8', newline='') as f:
        writer = csv.writer(f)
        writer.writerows(sanitized_rows)
    
    print(f"Sanitized schedule written to: {output_path}")
    print(f"Total rows written: {len(sanitized_rows)}")

if __name__ == '__main__':
    main()
