#!/usr/bin/env python
# -*- coding: UTF-8 -*-

from aip.ai_platform import AI_Platform


REQ_HOSTURL     = "http://10.72.1.22:32391"
REQ_USER = "admin"
REQ_PASSWORD = "sangfor123"

def push_end(res):
    print("push end:", res)

def create_ai_platform():
    AI_Platform(REQ_HOSTURL, REQ_USER, REQ_PASSWORD)
    AI_Platform(REQ_HOSTURL, "test1", "user@123")
    AI_Platform(REQ_HOSTURL, "test2", "user@123")

if __name__=="__main__":
    create_ai_platform()