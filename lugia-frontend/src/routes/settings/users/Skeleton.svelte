<div class="animate-pulse">
	<!-- Table Skeleton -->
	<div class="overflow-hidden shadow ring-1 ring-black/5 sm:rounded-lg">
		<div class="min-w-full divide-y divide-gray-300">
			<div class="bg-gray-50">
				<div class="grid grid-cols-5 gap-x-6 py-3.5 px-4 sm:px-6">
					<div class="h-4 bg-gray-300 rounded col-span-1"></div>
					<div class="h-4 bg-gray-300 rounded col-span-1"></div>
					<div class="h-4 bg-gray-300 rounded col-span-1"></div>
					<div class="h-4 bg-gray-300 rounded col-span-1"></div>
					<div class="h-4 bg-gray-300 rounded col-span-1"></div>
				</div>
			</div>
			<div class="divide-y divide-gray-200 bg-white">
				{#each Array(5)}
					<div class="grid grid-cols-5 gap-x-6 py-4 px-4 sm:px-6 items-center">
						<div class="h-4 bg-gray-300 rounded col-span-1"></div>
						<div class="h-6 bg-gray-300 rounded w-20 col-span-1"></div>
						<div class="h-4 bg-gray-300 rounded col-span-1"></div>
						<div class="h-4 bg-gray-300 rounded col-span-1"></div>
						<div class="h-4 bg-gray-300 rounded w-12 col-span-1 justify-self-end"></div>
					</div>
				{/each}
			</div>
		</div>
	</div>
</div>
