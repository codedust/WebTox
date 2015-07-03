(function() {
  var app = angular.module('fullscreen', []);

  app.service('Fullscreen', function() {
    this.goFullscreen = function() {
      var elem = document.querySelector("body");
      if (elem.requestFullscreen) {
        elem.requestFullscreen();
      } else if (elem.msRequestFullscreen) {
        elem.msRequestFullscreen();
      } else if (elem.mozRequestFullScreen) {
        elem.mozRequestFullScreen();
      } else if (elem.webkitRequestFullscreen) {
        elem.webkitRequestFullscreen();
      }
    };
  });
})();
