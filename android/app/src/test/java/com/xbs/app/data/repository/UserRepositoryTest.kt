package com.xbs.app.data.repository

import com.xbs.app.data.api.ErrorEnvelopeInterceptor
import com.xbs.app.data.api.UserApi
import retrofit2.converter.kotlinx.serialization.asConverterFactory
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import retrofit2.Retrofit

class UserRepositoryTest {
    private val server = MockWebServer()
    private lateinit var api: UserApi
    private lateinit var tokenStore: FakeTokenStore
    private lateinit var repo: UserRepository

    @Before
    fun setUp() {
        server.start()
        val json = Json { ignoreUnknownKeys = true }
        api = Retrofit.Builder()
            .baseUrl(server.url("/"))
            .client(OkHttpClient.Builder().addInterceptor(ErrorEnvelopeInterceptor()).build())
            .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
            .build()
            .create(UserApi::class.java)
        tokenStore = FakeTokenStore()
        repo = UserRepository(api, tokenStore)
    }

    @After fun tearDown() = server.shutdown()

    @Test
    fun `login success saves token`() = runTest {
        server.enqueue(MockResponse().setBody("""{"code":0,"message":"success","data":{"token":"t123"}}"""))
        val result = repo.login("alice", "password")
        assertTrue(result.isSuccess)
        assertEquals("t123", tokenStore.token())
        val recorded = server.takeRequest()
        assertEquals("POST", recorded.method)
        assertEquals("/users/login", recorded.path)
    }

    @Test
    fun `login wrong password returns ApiException in failure`() = runTest {
        server.enqueue(MockResponse().setResponseCode(401).setBody("""{"code":1002,"message":"wrong password"}"""))
        val result = repo.login("alice", "bad")
        assertTrue(result.isFailure)
        assertEquals("wrong password", result.exceptionOrNull()?.message)
        assertEquals(null, tokenStore.token())
    }
}
