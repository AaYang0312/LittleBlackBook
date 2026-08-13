package com.xbs.app.data.repository

import com.xbs.app.data.api.ErrorEnvelopeInterceptor
import com.xbs.app.data.api.NoteApi
import retrofit2.converter.kotlinx.serialization.asConverterFactory
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import retrofit2.Retrofit

class NoteRepositoryTest {
    private val server = MockWebServer()
    private lateinit var repo: NoteRepository

    @Before
    fun setUp() {
        server.start()
        val json = Json { ignoreUnknownKeys = true }
        val api = Retrofit.Builder()
            .baseUrl(server.url("/"))
            .client(OkHttpClient.Builder().addInterceptor(ErrorEnvelopeInterceptor()).build())
            .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
            .build()
            .create(NoteApi::class.java)
        repo = NoteRepository(api)
    }

    @After fun tearDown() = server.shutdown()

    @Test
    fun `latest parses page with next cursor`() = runTest {
        server.enqueue(MockResponse().setBody("""
            {"code":0,"message":"success","data":{"list":[
              {"id":100,"user_id":1,"title":"t","content":"c","cover_url":"u","images":["u"],
               "like_count":3,"collect_count":1,"comment_count":2,"created_at":"2026-08-12T10:00:00Z"}
            ],"next_cursor":100,"has_more":true}}
        """.trimIndent()))
        val page = repo.latest(null).getOrThrow()
        assertEquals(1, page.list.size)
        assertEquals(100L, page.nextCursor)
        assertTrue(page.hasMore)
        assertEquals(3L, page.list[0].likeCount)
        assertEquals("/notes/latest?size=20", server.takeRequest().path)
    }

    @Test
    fun `latest passes cursor on second page`() = runTest {
        server.enqueue(MockResponse().setBody("""{"code":0,"message":"success","data":{"list":[],"next_cursor":0,"has_more":false}}"""))
        val page = repo.latest(100).getOrThrow()
        assertFalse(page.hasMore)
        assertEquals("/notes/latest?cursor=100&size=20", server.takeRequest().path)
    }
}
