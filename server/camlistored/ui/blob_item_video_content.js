/*
Copyright 2014 The Camlistore Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

goog.provide('cam.BlobItemVideoContent');

goog.require('goog.math.Size');

// Renders video blob items. Currently recognizes movies by looking for a filename with a common movie extension.
cam.BlobItemVideoContent = React.createClass({
	displayName: 'BlobItemVideoContent',

	MIN_PREVIEW_SIZE: 128,

	propTypes: {
		isDescribed: React.PropTypes.bool.isRequired,
		aspect: React.PropTypes.number.isRequired,
		blobref: React.PropTypes.string.isRequired,
		filename: React.PropTypes.string.isRequired,
		href: React.PropTypes.string.isRequired,
		size: React.PropTypes.instanceOf(goog.math.Size).isRequired,
		posterSrc: React.PropTypes.string,
		src: React.PropTypes.string.isRequired,
	},

	getInitialState: function() {
		return {
			// loaded does not actually represent whether the video has loaded (enough).
			// It is rather an indicator that we switch from displaying the poster to
			// showing the video, and it is set to true by by clicking on the play button.
			loaded: false,
			posterLoaded: false,
			mouseover: false,
			playing: false,
		};
	},

	getThumbClipClassName_: function() {
		return React.addons.classSet({
			'cam-blobitem-thumbclip': true,
			'cam-blobitem-loading': false,
		});
	},

	render: function() {
		var thumbClipSize = new goog.math.Size(this.props.size.width, this.props.size.height);
		return React.DOM.div({
				className: React.addons.classSet({
					'cam-blobitem-video': true,
					'cam-blobitem-video-loaded': this.state.loaded,
				}),
				onMouseEnter: this.handleMouseOver_,
				onMouseLeave: this.handleMouseOut_,
			},
			React.DOM.a({href: this.props.href},
				// TODO(mpl): now that we have an image that can be used as a poster, wouldn't
				// it be simpler to just load the video element (with autoplay false oc), and set
				// its poster attribute with the image? (i.e. no more getPoster_()).
				this.getVideo_(),
				this.getCamera_(),
				this.getPoster_(thumbClipSize)
			),
			this.getPlayPauseButton_()
		);
	},

	getVideo_: function() {
		if (!this.state.loaded) {
			return null;
		}
		return React.DOM.video({
			autoPlay: true,
			src: goog.string.subs('%s%s/%s', goog.global.CAMLISTORE_CONFIG.downloadHelper, this.props.blobref, this.props.filename),
			width: this.props.size.width,
			height: this.props.size.height,
		})
	},

	getCamera_: function() {
		if (this.state.loaded) {
			return null;
		}
		if (this.state.posterLoaded) {
			return null;
		}
		return React.DOM.i({
			className: 'fa fa-video-camera',
			style: {
				fontSize: this.props.size.height / 1.5 + 'px',
				lineHeight: this.props.size.height + 'px',
				width: this.props.size.width,
			}
		})
	},

	getPoster_: function(thumbClipSize) {
		if (this.state.loaded) {
			return null;
		}
		if (!this.props.isDescribed) {
			// If server does not have ffmpeg, nothing was indexed about the video;
			// we just know it is a video.
			return null;
		}
		var thumbSize = this.getThumbSize_(thumbClipSize);
		var pos = cam.math.center(thumbSize, thumbClipSize);
		return React.DOM.img({
			className: 'cam-blobitem-thumb',
			onLoad: this.onThumbLoad_,
			src: this.props.posterSrc,
			style: {left:pos.x, top:pos.y, visibility:(this.state.posterLoaded ? 'visible' : 'hidden')},
			title: this.props.filename,
			width: this.props.size.width,
			height: this.props.size.height,
		})
	},

	onThumbLoad_: function() {
		this.setState({posterLoaded:true});
	},

	getThumbSize_: function(thumbClipSize) {
		var bleed = true;
		return cam.math.scaleToFit(new goog.math.Size(this.props.aspect, 1), thumbClipSize, bleed);
	},

	getPlayPauseButton_: function() {
		if (!this.state.mouseover || this.props.size.width < this.MIN_PREVIEW_SIZE || this.props.size.height < this.MIN_PREVIEW_SIZE) {
			return null;
		}
		return React.DOM.i({
			className: React.addons.classSet({
					'fa': true,
					'fa-play': !this.state.playing,
					'fa-pause': this.state.playing,
				}),
			onClick: this.handlePlayPauseClick_,
			style: {
				fontSize: this.props.size.height / 5 + 'px',
			}
		})
	},

	handlePlayPauseClick_: function(e) {
		this.setState({
			loaded: true,
			playing: !this.state.playing,
		});

		if (this.state.loaded) {
			var video = this.getDOMNode().querySelector('video');
			if (this.state.playing) {
				video.pause();
			} else {
				video.play();
			}
		}
	},

	handleMouseOver_: function() {
		this.setState({mouseover:true});
	},

	handleMouseOut_: function() {
		this.setState({mouseover:false});
	},
});

cam.BlobItemVideoContent.isVideo = function(rm) {
	if (rm && rm.video) {
		return true;
	}
	return false;
};

cam.BlobItemVideoContent.getHandler = function(blobref, searchSession, href) {
	var rm = searchSession.getResolvedMeta(blobref);
	if (!rm || !rm.video) {
		return null;
	}
	return new cam.BlobItemVideoContent.Handler(rm, href);
};

cam.BlobItemVideoContent.Handler = function(rm, href) {
	this.rm_ = rm;
	this.href_ = href;
	this.thumber_ = cam.Thumber.fromVideoMeta(rm);
};

cam.BlobItemVideoContent.Handler.prototype.getAspectRatio = function() {
	if (this.rm_.video.height == 0) {
		return 1;
	}
	return this.rm_.video.width / this.rm_.video.height;
};

cam.BlobItemVideoContent.Handler.prototype.createContent = function(size) {
	return cam.BlobItemVideoContent({
		isDescribed: (this.rm_.video && this.rm_.video.height != 0),
		aspect: this.getAspectRatio(),
		blobref: this.rm_.blobRef,
		filename: this.rm_.file.fileName,
		href: this.href_,
		size: size,
		posterSrc: this.thumber_.getSrc(size.height),
		src: goog.string.subs('%s%s/%s', goog.global.CAMLISTORE_CONFIG.downloadHelper, this.rm_.blobRef, this.rm_.file.fileName),
	});
};
