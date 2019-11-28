(function () {

  "use strict";

  var App = angular.module("App.directives", []);

  App.directive('inputtext', function ($timeout) {
    return {
      restrict: 'E',
      replace: true,
      template: '<input type="text"/>',
      scope: {
        //if there were attributes it would be shown here
      },
      link: function (scope, element, attrs, ctrl) {
        // DOM manipulation may happen here.
      }
    }
  });

  App.directive('version', function (version) {
    return function (scope, elm, attrs) {
      elm.text(version);
    };
  });


  // https://stackoverflow.com/questions/12592472/how-to-highlight-a-current-menu-item
  App.directive('activeLink', ['$location', function (location) {
    return {
      restrict: 'A',
      link: function (scope, element, attrs, controller) {
        var clazz = attrs.activeLink;
        var path = attrs.href;
        path = path.substring(2); //hack because path does not return including hashbang
        scope.location = location;
        scope.$watch('location.path()', function (newPath) {
          if (path === newPath) {
            element.addClass(clazz);
          } else {
            element.removeClass(clazz);
          }
        });
      }
    };
  }]);

  // you may add as much directives as you want below
}());