#!/bin/bash
# SeaweedFS 초기화 스크립트

set -e

echo "🌱 SeaweedFS 초기화를 시작합니다..."

# SeaweedFS 데이터 디렉토리 초기화
echo "📁 SeaweedFS 데이터 디렉토리를 초기화합니다..."
rm -rf /home/pot-play-storage/deploy/data/seaweedfs/master/*
rm -rf /home/pot-play-storage/deploy/data/seaweedfs/volume/*
rm -rf /home/pot-play-storage/deploy/data/seaweedfs/filer/*

# 디렉토리 재생성
mkdir -p /home/pot-play-storage/deploy/data/seaweedfs/master
mkdir -p /home/pot-play-storage/deploy/data/seaweedfs/volume
mkdir -p /home/pot-play-storage/deploy/data/seaweedfs/filer

# 권한 설정
chmod -R 777 /home/pot-play-storage/deploy/data/seaweedfs

echo "✅ SeaweedFS 초기화가 완료되었습니다!"