/*
** Zabbix
** Copyright (C) 2001-2020 Zabbix SIA
**
** This program is free software; you can redistribute it and/or modify
** it under the terms of the GNU General Public License as published by
** the Free Software Foundation; either version 2 of the License, or
** (at your option) any later version.
**
** This program is distributed in the hope that it will be useful,
** but WITHOUT ANY WARRANTY; without even the implied warranty of
** MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
** GNU General Public License for more details.
**
** You should have received a copy of the GNU General Public License
** along with this program; if not, write to the Free Software
** Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301, USA.
**/


#ifndef ZABBIX_DIAG_H
#define ZABBIX_DIAG_H

#include "common.h"
#include "zbxjson.h"

#define ZBX_DIAG_SECTION_MAX	64
#define ZBX_DIAG_FIELD_MAX	64

#define ZBX_DIAG_STATS_ALL			0xFFFFFFFF

#define ZBX_DIAG_HISTORYCACHE_ITEMS		0x00000001
#define ZBX_DIAG_HISTORYCACHE_VALUES		0x00000002
#define ZBX_DIAG_HISTORYCACHE_MEM_DATA		0x00000004
#define ZBX_DIAG_HISTORYCACHE_MEM_INDEX		0x00000008
#define ZBX_DIAG_HISTORYCACHE_MEM_TRENDS	0x00000010

#define ZBX_DIAG_HISTORYCACHE_SIMPLE	(ZBX_DIAG_HISTORYCACHE_ITEMS | \
					ZBX_DIAG_HISTORYCACHE_VALUES)

#define ZBX_DIAG_HISTORYCACHE_MEM		(ZBX_DIAG_HISTORYCACHE_MEM_DATA | \
					ZBX_DIAG_HISTORYCACHE_MEM_INDEX | \
					ZBX_DIAG_HISTORYCACHE_MEM_TRENDS)

typedef struct
{
	char		*name;
	zbx_uint64_t	value;
}
zbx_diag_map_t;

int	diag_add_section_info(const char *section, const struct zbx_json_parse *jp, struct zbx_json *j,
		char **error);

int	diag_add_historycache_info(const struct zbx_json_parse *jp, const zbx_diag_map_t *stats, struct zbx_json *j,
		char **error);

#endif
