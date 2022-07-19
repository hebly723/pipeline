#!/usr/bin/env python
# -*- coding: UTF-8 -*-

from aip.ai_platform import AI_Platform


REQ_HOSTURL     = "http://10.72.1.22:32391"
REQ_USER = "admin"
REQ_PASSWORD = "sangfor123"

def push_end(res):
    print("push end:", res)

schema = [ {
        "name": "image_md5",
        "type": "string",
        "comment": "图像文件的MD5值",
        "nullable": False
    }, {
        "name": "image_fname",
        "type": "string",
        "comment": "图像文件的文件名",
        "nullable": False
    }, {
        "name": "labels",
        "type": "string",
        "comment": "文件的标注数据",
        "nullable": True
    }]



def create_ai_platform():
    return AI_Platform(REQ_HOSTURL, REQ_USER, REQ_PASSWORD)
if __name__=="__main__":
    aip = create_ai_platform()
    ds = aip.create_structured_dataset(dataspace="9d30",dataset="edr_21", data_schema=schema)
    ds.add_data(path="labels.csv", format="csv", header=False)
    ds.push(push_end)
    df = ds.get(sql="select * from data_table where image_fname = \"images/1.png\"", limit = 1000) # return pd.DataFrame
    aip.wait_done()