export function getCroppedImg(
	imageSrc: string,
	pixelCrop: { x: number; y: number; width: number; height: number },
	outputSize: number = 280
): Promise<Blob> {
	return new Promise((resolve, reject) => {
		const image = new Image();
		image.src = imageSrc;
		image.onload = () => {
			const canvas = document.createElement('canvas');
			canvas.width = outputSize;
			canvas.height = outputSize;
			const ctx = canvas.getContext('2d');

			if (!ctx) {
				reject(new Error('No 2d context'));
				return;
			}

			ctx.imageSmoothingEnabled = true;
			ctx.imageSmoothingQuality = 'high';

			ctx.drawImage(
				image,
				pixelCrop.x,
				pixelCrop.y,
				pixelCrop.width,
				pixelCrop.height,
				0,
				0,
				outputSize,
				outputSize
			);

			canvas.toBlob(
				(blob) => {
					if (!blob) {
						reject(new Error('Canvas is empty'));
						return;
					}
					resolve(blob);
				},
				'image/jpeg',
				0.9
			);
		};
		image.onerror = () => {
			reject(new Error('Failed to load image for cropping'));
		};
	});
}
