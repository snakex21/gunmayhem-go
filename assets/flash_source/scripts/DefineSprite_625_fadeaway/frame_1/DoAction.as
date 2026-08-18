_X = - _root._x;
_Y = - _root._y;
if(_root.gamewin)
{
   _X = _root.activeplayers[0]._x - 225;
   _Y = _root.activeplayers[0]._y - 200;
   _xscale = 50;
   _yscale = 50;
}
this.onEnterFrame = function()
{
   _X = - _root._x;
   _Y = - _root._y;
   if(_root.gamewin)
   {
      _X = _root.activeplayers[0]._x - 225;
      _Y = _root.activeplayers[0]._y - 200;
   }
};
