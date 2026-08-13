package com.xbs.app

/** 轮询等待条件成立；MockWebServer 在本机回包极快，3s 上限足够。 */
fun awaitUntil(timeoutMs: Long = 3000, condition: () -> Boolean) {
    val deadline = System.currentTimeMillis() + timeoutMs
    while (!condition()) {
        if (System.currentTimeMillis() > deadline) throw AssertionError("condition not met within ${timeoutMs}ms")
        Thread.sleep(20)
    }
}
